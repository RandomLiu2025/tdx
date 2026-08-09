package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/injoyai/tdx/protocol"
)

func TestQuerySince(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantEnabled bool
		wantDate    string
		wantErr     bool
	}{
		{name: "empty"},
		{name: "valid", query: "?since=20260808", wantEnabled: true, wantDate: "20260808"},
		{name: "invalid format", query: "?since=2026-08-08", wantErr: true},
		{name: "invalid date", query: "?since=20260230", wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/"+testCase.query, nil)
			got, enabled, err := querySince(request)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("querySince() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("querySince() unexpected error: %v", err)
			}
			if enabled != testCase.wantEnabled {
				t.Fatalf("querySince() enabled = %t, want %t", enabled, testCase.wantEnabled)
			}
			if enabled && got.Format("20060102") != testCase.wantDate {
				t.Errorf("querySince() date = %s, want %s", got.Format("20060102"), testCase.wantDate)
			}
		})
	}
}

func TestQueryKlineAdjustment(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    klineAdjustment
		wantErr bool
	}{
		{name: "default", want: adjustmentNone},
		{name: "none", query: "?adjust=none", want: adjustmentNone},
		{name: "qfq case insensitive", query: "?adjust=QFQ", want: adjustmentQFQ},
		{name: "hfq", query: "?adjust=hfq", want: adjustmentHFQ},
		{name: "unsupported", query: "?adjust=forward", wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/"+testCase.query, nil)
			got, err := queryKlineAdjustment(request)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("queryKlineAdjustment() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("queryKlineAdjustment() unexpected error: %v", err)
			}
			if got != testCase.want {
				t.Errorf("queryKlineAdjustment() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestFilterKlinesSince(t *testing.T) {
	since := time.Date(2026, 8, 8, 0, 0, 0, 0, time.Local)
	response := &protocol.KlineResp{
		Count: 3,
		List: []*protocol.Kline{
			{Time: since.AddDate(0, 0, -1)},
			{Time: since},
			{Time: since.AddDate(0, 0, 1)},
		},
	}

	filterKlinesSince(response, since)

	if response.Count != 2 {
		t.Fatalf("filterKlinesSince() count = %d, want 2", response.Count)
	}
	if len(response.List) != 2 {
		t.Fatalf("filterKlinesSince() list length = %d, want 2", len(response.List))
	}
	if !response.List[0].Time.Equal(since) {
		t.Errorf("filterKlinesSince() first time = %s, want %s", response.List[0].Time, since)
	}
}

func TestApplyKlineAdjustment(t *testing.T) {
	dayOne := time.Date(2024, 1, 1, 15, 0, 0, 0, time.Local)
	dayTwo := time.Date(2024, 1, 2, 15, 0, 0, 0, time.Local)
	gbbq := &protocol.GbbqResp{
		Count: 1,
		List: []*protocol.Gbbq{
			{Code: "sh600000", Time: dayTwo, Category: 1, C1: 10},
		},
	}
	newResponse := func() *protocol.KlineResp {
		return &protocol.KlineResp{
			Count: 2,
			List: []*protocol.Kline{
				{Time: dayOne, Last: 10000, Open: 10000, High: 10000, Low: 10000, Close: 10000},
				{Time: dayTwo, Last: 9000, Open: 9000, High: 9000, Low: 9000, Close: 9000},
			},
		}
	}

	qfq := newResponse()
	applyKlineAdjustment(qfq, gbbq, adjustmentQFQ)
	if qfq.List[0].Close != 9000 || qfq.List[1].Close != 9000 {
		t.Errorf("QFQ closes = [%v %v], want [9.000 9.000]", qfq.List[0].Close, qfq.List[1].Close)
	}

	hfq := newResponse()
	applyKlineAdjustment(hfq, gbbq, adjustmentHFQ)
	if hfq.List[0].Close != 10000 || hfq.List[1].Close != 10000 {
		t.Errorf("HFQ closes = [%v %v], want [10.000 10.000]", hfq.List[0].Close, hfq.List[1].Close)
	}
}

func TestAdjustAndFilterKlinesUsesFullHistoryAnchor(t *testing.T) {
	dayOne := time.Date(2024, 1, 1, 15, 0, 0, 0, time.Local)
	dayTwo := time.Date(2024, 1, 2, 15, 0, 0, 0, time.Local)
	response := &protocol.KlineResp{
		Count: 2,
		List: []*protocol.Kline{
			{Time: dayOne, Last: 10000, Open: 10000, High: 10000, Low: 10000, Close: 10000},
			{Time: dayTwo, Last: 9000, Open: 9000, High: 9000, Low: 9000, Close: 9000},
		},
	}
	gbbq := &protocol.GbbqResp{
		Count: 1,
		List: []*protocol.Gbbq{
			{Code: "sh600000", Time: dayTwo, Category: 1, C1: 10},
		},
	}

	adjustAndFilterKlines(response, gbbq, adjustmentHFQ, dayTwo, true)

	if response.Count != 1 || len(response.List) != 1 {
		t.Fatalf("adjustAndFilterKlines() count = %d, length = %d, want 1", response.Count, len(response.List))
	}
	if response.List[0].Close != 10000 {
		t.Errorf("filtered HFQ close = %v, want 10.000 from full-history anchor", response.List[0].Close)
	}
}

func TestKlineRoutesValidateNewParameters(t *testing.T) {
	apiServer := &Server{}
	mux := http.NewServeMux()
	apiServer.registerRoutes(mux)

	paths := []string{
		"/kline/minute/241",
		"/kline/minute/241?code=sz000001&since=invalid",
		"/kline/all?type=9&code=sh600519&since=invalid",
		"/kline/day/all?code=sh600519&adjust=invalid",
		"/index/all?type=9&code=sh000001&since=invalid",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("GET %s status = %d, want %d", path, recorder.Code, http.StatusBadRequest)
			}
		})
	}
}
