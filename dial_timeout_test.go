package tdx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRangeDialHonorsCanceledContext(test *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dial := NewRangeDialWithTimeout([]string{"127.0.0.1:1"}, time.Second)
	connection, _, err := dial(ctx)
	if connection != nil {
		_ = connection.Close()
		test.Fatal("dial returned a connection for canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		test.Fatalf("dial error = %v, want context.Canceled", err)
	}
}
