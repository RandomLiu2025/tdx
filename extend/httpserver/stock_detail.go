package httpserver

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/injoyai/tdx/protocol"
)

const stockDetailCacheTTL = 30 * time.Minute

const (
	blockCategoryConcept  = "concept"
	blockCategoryIndustry = "industry"
	blockCategoryStyle    = "style"
	blockCategoryIndex    = "index"
	blockCategoryGeneral  = "general"
	blockCategorySpecial  = "special"
)

type StockDetail struct {
	Code     string        `json:"code"`
	Name     string        `json:"name"`
	Blocks   []StockBlock  `json:"blocks"`
	Industry StockIndustry `json:"industry"`
}

type StockBlock struct {
	Name     string `json:"name"`
	Index    string `json:"index"`
	Category string `json:"category"`
}

type StockIndustry struct {
	TdxCode string `json:"tdxCode"`
	SwCode  string `json:"swCode"`
}

type stockDetailSource interface {
	GetCodeAll(exchange protocol.Exchange) (*protocol.CodeResp, error)
	GetBlockData(file string) ([]*protocol.Block, error)
	GetZHBFiles() (map[string][]byte, error)
	GetTdxHy() ([]*protocol.TdxHy, error)
}

type stockDetailCache struct {
	mu        sync.Mutex
	details   []StockDetail
	expiresAt time.Time
}

type stockBlockSource struct {
	file     string
	category string
}

var stockBlockSources = []stockBlockSource{
	{file: protocol.BlockFileGN, category: blockCategoryConcept},
	{file: protocol.BlockFileHY, category: blockCategoryIndustry},
	{file: protocol.BlockFileFG, category: blockCategoryStyle},
	{file: protocol.BlockFileZS, category: blockCategoryIndex},
	{file: protocol.BlockFile, category: blockCategoryGeneral},
}

var stockBlockCategoryOrder = map[string]int{
	blockCategoryConcept:  0,
	blockCategoryIndustry: 1,
	blockCategoryStyle:    2,
	blockCategoryIndex:    3,
	blockCategoryGeneral:  4,
	blockCategorySpecial:  5,
}

func loadStockDetails(source stockDetailSource) ([]StockDetail, error) {
	details := make(map[string]*StockDetail)
	for _, exchange := range []protocol.Exchange{protocol.ExchangeSH, protocol.ExchangeSZ, protocol.ExchangeBJ} {
		response, err := source.GetCodeAll(exchange)
		if err != nil {
			return nil, fmt.Errorf("获取%s市场代码失败: %w", exchange.Name(), err)
		}
		for _, item := range response.List {
			code := exchange.String() + item.Code
			if protocol.IsStock(code) {
				details[code] = &StockDetail{Code: code, Name: item.Name, Blocks: []StockBlock{}}
			}
		}
	}

	files, err := source.GetZHBFiles()
	if err != nil {
		return nil, fmt.Errorf("获取板块配置总包失败: %w", err)
	}
	zsData, err := stockDetailFile(files, protocol.FileTdxZs)
	if err != nil {
		return nil, err
	}
	bkData, err := stockDetailFile(files, protocol.FileTdxBk)
	if err != nil {
		return nil, err
	}
	spData, err := stockDetailFile(files, protocol.FileSpBlock)
	if err != nil {
		return nil, err
	}
	zs := protocol.ParseTdxZs(zsData)
	bk := protocol.ParseTdxBk(bkData)
	blockIndexes := stockBlockIndexes(zs, bk)
	seen := make(map[string]map[string]struct{}, len(details))

	for _, blockSource := range stockBlockSources {
		blocks, err := source.GetBlockData(blockSource.file)
		if err != nil {
			return nil, fmt.Errorf("获取%s板块数据失败: %w", blockSource.category, err)
		}
		protocol.FillBlockIndexAlias(blocks, zs, bk)
		for _, block := range blocks {
			membership := StockBlock{Name: block.Name, Index: block.Index, Category: blockSource.category}
			for _, rawCode := range block.Codes {
				appendStockBlock(details, seen, rawCode, membership)
			}
		}
	}

	for _, block := range protocol.ParseSpBlock(spData) {
		membership := StockBlock{
			Name:     block.Name,
			Index:    blockIndexes[block.Name],
			Category: blockCategorySpecial,
		}
		for _, rawCode := range block.Codes {
			appendStockBlock(details, seen, rawCode, membership)
		}
	}

	industries, err := source.GetTdxHy()
	if err != nil {
		return nil, fmt.Errorf("获取行业归属失败: %w", err)
	}
	for _, industry := range industries {
		code := protocol.Exchange(industry.Market).String() + industry.Code
		if detail := details[code]; detail != nil {
			detail.Industry = StockIndustry{TdxCode: industry.TdxHy, SwCode: industry.SwHy}
		}
	}

	result := make([]StockDetail, 0, len(details))
	for _, detail := range details {
		sort.Slice(detail.Blocks, func(i, j int) bool {
			left := detail.Blocks[i]
			right := detail.Blocks[j]
			if stockBlockCategoryOrder[left.Category] != stockBlockCategoryOrder[right.Category] {
				return stockBlockCategoryOrder[left.Category] < stockBlockCategoryOrder[right.Category]
			}
			if left.Name != right.Name {
				return left.Name < right.Name
			}
			return left.Index < right.Index
		})
		result = append(result, *detail)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result, nil
}

func stockDetailFile(files map[string][]byte, name string) ([]byte, error) {
	data, ok := files[name]
	if !ok {
		return nil, fmt.Errorf("%s 中缺少 %s", protocol.ReportZHB, name)
	}
	return data, nil
}

func stockBlockIndexes(zs []*protocol.TdxZs, bk []*protocol.TdxBk) map[string]string {
	indexes := make(map[string]string, len(zs)+len(bk))
	for _, item := range zs {
		indexes[item.Name] = item.Code
	}
	for _, alias := range bk {
		if index := indexes[alias.Full]; index != "" {
			indexes[alias.Short] = index
		}
	}
	return indexes
}

func appendStockBlock(details map[string]*StockDetail, seen map[string]map[string]struct{}, rawCode string, block StockBlock) {
	code, ok := fullStockCode(rawCode)
	if !ok {
		return
	}
	detail := details[code]
	if detail == nil {
		return
	}
	if seen[code] == nil {
		seen[code] = make(map[string]struct{})
	}
	key := block.Category + "\x00" + block.Index + "\x00" + block.Name
	if _, exists := seen[code][key]; exists {
		return
	}
	seen[code][key] = struct{}{}
	detail.Blocks = append(detail.Blocks, block)
}

func fullStockCode(rawCode string) (string, bool) {
	if len(rawCode) != 7 {
		return "", false
	}
	var exchange protocol.Exchange
	switch rawCode[0] {
	case '0':
		exchange = protocol.ExchangeSZ
	case '1':
		exchange = protocol.ExchangeSH
	case '2':
		exchange = protocol.ExchangeBJ
	default:
		return "", false
	}
	return exchange.String() + rawCode[1:], true
}

func (cache *stockDetailCache) getOrLoad(now time.Time, loader func() ([]StockDetail, error)) ([]StockDetail, error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if now.Before(cache.expiresAt) {
		return cache.details, nil
	}
	details, err := loader()
	if err != nil {
		return nil, err
	}
	cache.details = details
	cache.expiresAt = now.Add(stockDetailCacheTTL)
	return cache.details, nil
}
