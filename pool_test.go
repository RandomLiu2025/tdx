package tdx

import "testing"

func TestPoolReady(test *testing.T) {
	pool := &Pool{}
	if pool.Ready() {
		test.Fatal("empty pool should not be ready")
	}

	client := &Client{}
	pool.clients = []*Client{client}
	if pool.Ready() {
		test.Fatal("disconnected client should not make pool ready")
	}

	client.connected.Store(true)
	if !pool.Ready() {
		test.Fatal("connected client should make pool ready")
	}
}
