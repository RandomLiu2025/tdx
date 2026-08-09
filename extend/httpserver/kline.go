package httpserver

import (
	"time"

	"github.com/injoyai/tdx"
	"github.com/injoyai/tdx/protocol"
)

func loadKlineAll(c *tdx.Client, typ uint8, code string, since time.Time, bounded bool) (*protocol.KlineResp, error) {
	if !bounded {
		return c.GetKlineAll(typ, code)
	}
	response, err := c.GetKlineUntil(typ, code, func(kline *protocol.Kline) bool {
		return kline.Time.Before(since)
	})
	if err != nil {
		return nil, err
	}
	filterKlinesSince(response, since)
	return response, nil
}

func loadIndexAll(c *tdx.Client, typ uint8, code string, since time.Time, bounded bool) (*protocol.KlineResp, error) {
	if !bounded {
		return c.GetIndexAll(typ, code)
	}
	response, err := c.GetIndexUntil(typ, code, func(kline *protocol.Kline) bool {
		return kline.Time.Before(since)
	})
	if err != nil {
		return nil, err
	}
	filterKlinesSince(response, since)
	return response, nil
}

func filterKlinesSince(response *protocol.KlineResp, since time.Time) {
	filtered := make([]*protocol.Kline, 0, len(response.List))
	for _, kline := range response.List {
		if !kline.Time.Before(since) {
			filtered = append(filtered, kline)
		}
	}
	response.List = filtered
	response.Count = uint16(len(filtered))
}

func applyKlineAdjustment(response *protocol.KlineResp, gbbq *protocol.GbbqResp, adjustment klineAdjustment) {
	if adjustment == adjustmentNone || len(response.List) == 0 {
		return
	}

	events := make(protocol.XRXDs, 0, len(gbbq.List))
	for _, item := range gbbq.List {
		if item.IsXRXD() {
			events = append(events, item.XRXD())
		}
	}
	factors := events.Pre(response.List).Factors()
	switch adjustment {
	case adjustmentQFQ:
		response.List = protocol.ApplyQFQ(response.List, factors)
	case adjustmentHFQ:
		response.List = protocol.ApplyHFQ(response.List, factors)
	}
}

func adjustAndFilterKlines(response *protocol.KlineResp, gbbq *protocol.GbbqResp, adjustment klineAdjustment, since time.Time, bounded bool) {
	applyKlineAdjustment(response, gbbq, adjustment)
	if bounded {
		filterKlinesSince(response, since)
	}
}
