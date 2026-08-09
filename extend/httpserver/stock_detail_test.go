package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/injoyai/tdx/protocol"
)

type fakeStockDetailSource struct {
	codes      map[protocol.Exchange]*protocol.CodeResp
	blocks     map[string][]*protocol.Block
	files      map[string][]byte
	industries []*protocol.TdxHy
}

func (source *fakeStockDetailSource) GetCodeAll(exchange protocol.Exchange) (*protocol.CodeResp, error) {
	response, ok := source.codes[exchange]
	if !ok {
		return nil, errors.New("missing code response")
	}
	return response, nil
}

func (source *fakeStockDetailSource) GetBlockData(file string) ([]*protocol.Block, error) {
	blocks, ok := source.blocks[file]
	if !ok {
		return nil, errors.New("missing block response")
	}
	return blocks, nil
}

func (source *fakeStockDetailSource) GetZHBFiles() (map[string][]byte, error) {
	return source.files, nil
}

func (source *fakeStockDetailSource) GetTdxHy() ([]*protocol.TdxHy, error) {
	return source.industries, nil
}

func TestLoadStockDetails(t *testing.T) {
	source := &fakeStockDetailSource{
		codes: map[protocol.Exchange]*protocol.CodeResp{
			protocol.ExchangeSH: {
				List: []*protocol.Code{
					{Code: "600000", Name: "Shanghai Stock"},
					{Code: "510300", Name: "ETF"},
				},
			},
			protocol.ExchangeSZ: {
				List: []*protocol.Code{
					{Code: "000001", Name: "Shenzhen Stock"},
					{Code: "399001", Name: "Index"},
				},
			},
			protocol.ExchangeBJ: {
				List: []*protocol.Code{
					{Code: "920001", Name: "Beijing Stock"},
				},
			},
		},
		blocks: map[string][]*protocol.Block{
			protocol.BlockFileGN: {
				{Name: "Concept", Codes: []string{"1600000", "1600000", "0000001"}},
			},
			protocol.BlockFileHY: {
				{Name: "Bank", Index: "880002", Codes: []string{"1600000"}},
			},
			protocol.BlockFileFG: {
				{Name: "Style", Index: "880003", Codes: []string{"1600000"}},
			},
			protocol.BlockFileZS: {
				{Name: "Main Index", Index: "000300", Codes: []string{"1600000"}},
			},
			protocol.BlockFile: {
				{Name: "General", Index: "880004", Codes: []string{"1600000", "invalid"}},
			},
		},
		files: map[string][]byte{
			protocol.FileTdxZs:   []byte("Concept Full|880001|1|1|0|Concept Full\nLarge Index|000905|1|1|0|Large Index\n"),
			protocol.FileTdxBk:   []byte("1|Concept|Concept Full|0\n"),
			protocol.FileSpBlock: []byte("#Large Index\r\n1600000\r\n2920001\r\n"),
		},
		industries: []*protocol.TdxHy{
			{Market: protocol.ExchangeSH.Uint8(), Code: "600000", TdxHy: "T01", SwHy: "X01"},
			{Market: protocol.ExchangeSZ.Uint8(), Code: "000001", TdxHy: "T02", SwHy: "X02"},
			{Market: protocol.ExchangeBJ.Uint8(), Code: "920001", TdxHy: "T03", SwHy: "X03"},
		},
	}

	details, err := loadStockDetails(source)
	if err != nil {
		t.Fatalf("loadStockDetails() unexpected error: %v", err)
	}
	if len(details) != 3 {
		t.Fatalf("loadStockDetails() length = %d, want 3", len(details))
	}
	wantCodes := []string{"bj920001", "sh600000", "sz000001"}
	for index, want := range wantCodes {
		if details[index].Code != want {
			t.Errorf("details[%d].Code = %q, want %q", index, details[index].Code, want)
		}
	}

	shanghai := findStockDetail(details, "sh600000")
	if shanghai == nil {
		t.Fatal("missing sh600000 detail")
	}
	if shanghai.Name != "Shanghai Stock" {
		t.Errorf("sh600000 name = %q, want %q", shanghai.Name, "Shanghai Stock")
	}
	if shanghai.Industry.TdxCode != "T01" || shanghai.Industry.SwCode != "X01" {
		t.Errorf("sh600000 industry = %+v", shanghai.Industry)
	}
	if len(shanghai.Blocks) != 6 {
		t.Fatalf("sh600000 block count = %d, want 6", len(shanghai.Blocks))
	}
	wantBlocks := map[string]StockBlock{
		blockCategoryConcept:  {Name: "Concept", Index: "880001", Category: blockCategoryConcept},
		blockCategoryIndustry: {Name: "Bank", Index: "880002", Category: blockCategoryIndustry},
		blockCategoryStyle:    {Name: "Style", Index: "880003", Category: blockCategoryStyle},
		blockCategoryIndex:    {Name: "Main Index", Index: "000300", Category: blockCategoryIndex},
		blockCategoryGeneral:  {Name: "General", Index: "880004", Category: blockCategoryGeneral},
		blockCategorySpecial:  {Name: "Large Index", Index: "000905", Category: blockCategorySpecial},
	}
	for _, block := range shanghai.Blocks {
		want, ok := wantBlocks[block.Category]
		if !ok {
			t.Errorf("unexpected block category %q", block.Category)
			continue
		}
		if block != want {
			t.Errorf("block %q = %+v, want %+v", block.Category, block, want)
		}
	}

	beijing := findStockDetail(details, "bj920001")
	if beijing == nil || len(beijing.Blocks) != 1 || beijing.Blocks[0].Category != blockCategorySpecial {
		t.Errorf("bj920001 special block mapping = %+v", beijing)
	}
}

func TestStockDetailJSONUsesLowerCamelCase(t *testing.T) {
	data, err := json.Marshal(StockDetail{
		Code: "sh600000",
		Name: "Stock",
		Blocks: []StockBlock{
			{Name: "Bank", Index: "880001", Category: blockCategoryIndustry},
		},
		Industry: StockIndustry{TdxCode: "T01", SwCode: "X01"},
	})
	if err != nil {
		t.Fatalf("json.Marshal() unexpected error: %v", err)
	}
	want := `{"code":"sh600000","name":"Stock","blocks":[{"name":"Bank","index":"880001","category":"industry"}],"industry":{"tdxCode":"T01","swCode":"X01"}}`
	if string(data) != want {
		t.Errorf("StockDetail JSON = %s, want %s", data, want)
	}
}

func TestStockDetailCache(t *testing.T) {
	var cache stockDetailCache
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.Local)
	loadCount := 0
	loader := func() ([]StockDetail, error) {
		loadCount++
		return []StockDetail{{Code: "sh600000"}}, nil
	}

	first, err := cache.getOrLoad(now, loader)
	if err != nil || len(first) != 1 {
		t.Fatalf("first getOrLoad() = %+v, %v", first, err)
	}
	second, err := cache.getOrLoad(now.Add(10*time.Minute), loader)
	if err != nil || len(second) != 1 {
		t.Fatalf("cached getOrLoad() = %+v, %v", second, err)
	}
	if loadCount != 1 {
		t.Fatalf("cache load count = %d, want 1", loadCount)
	}

	_, err = cache.getOrLoad(now.Add(stockDetailCacheTTL+time.Second), loader)
	if err != nil {
		t.Fatalf("expired getOrLoad() unexpected error: %v", err)
	}
	if loadCount != 2 {
		t.Fatalf("expired cache load count = %d, want 2", loadCount)
	}
}

func TestStockDetailCacheFailureDoesNotReplaceSnapshot(t *testing.T) {
	var cache stockDetailCache
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.Local)
	_, err := cache.getOrLoad(now, func() ([]StockDetail, error) {
		return []StockDetail{{Code: "sh600000"}}, nil
	})
	if err != nil {
		t.Fatalf("initial getOrLoad() unexpected error: %v", err)
	}

	_, err = cache.getOrLoad(now.Add(stockDetailCacheTTL+time.Second), func() ([]StockDetail, error) {
		return nil, errors.New("load failed")
	})
	if err == nil {
		t.Fatal("expired getOrLoad() expected error")
	}
	if len(cache.details) != 1 || cache.details[0].Code != "sh600000" {
		t.Errorf("failed load replaced cached snapshot: %+v", cache.details)
	}
}

func TestStockDetailCacheCoalescesConcurrentLoads(t *testing.T) {
	var cache stockDetailCache
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.Local)
	var loadCount atomic.Int32
	loader := func() ([]StockDetail, error) {
		loadCount.Add(1)
		time.Sleep(10 * time.Millisecond)
		return []StockDetail{{Code: "sh600000"}}, nil
	}

	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, 8)
	for range 8 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			_, err := cache.getOrLoad(now, loader)
			errorsChannel <- err
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Errorf("concurrent getOrLoad() unexpected error: %v", err)
		}
	}
	if loadCount.Load() != 1 {
		t.Errorf("concurrent cache load count = %d, want 1", loadCount.Load())
	}
}

func TestStockDetailRouteRegistered(t *testing.T) {
	apiServer := &Server{}
	mux := http.NewServeMux()
	apiServer.registerRoutes(mux)

	request := httptest.NewRequest(http.MethodGet, "/code/stocks/detail", nil)
	_, pattern := mux.Handler(request)
	if pattern != "GET /code/stocks/detail" {
		t.Errorf("route pattern = %q, want %q", pattern, "GET /code/stocks/detail")
	}
}

func findStockDetail(details []StockDetail, code string) *StockDetail {
	for index := range details {
		if details[index].Code == code {
			return &details[index]
		}
	}
	return nil
}
