package tdx

import (
	"context"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/injoyai/ios"
	"github.com/injoyai/ios/module/tcp"
	"github.com/injoyai/logs"
)

func NewTCPDial(addr string) ios.DialFunc {
	if !strings.Contains(addr, ":") {
		addr += ":7709"
	}
	return tcp.NewDial(addr)
}

func NewHostDial(hosts []string) ios.DialFunc {
	if len(hosts) == 0 {
		hosts = Hosts
	}
	index := 0

	return func(ctx context.Context) (ios.ReadWriteCloser, string, error) {
		defer func() { index++ }()
		if index >= len(hosts) {
			index = 0
		}
		addr := hosts[index]
		if !strings.Contains(addr, ":") {
			addr += ":7709"
		}
		c, err := net.Dial("tcp", addr)
		return c, addr, err
	}
}

func NewRandomDial(hosts []string) ios.DialFunc {
	if len(hosts) == 0 {
		hosts = Hosts
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return func(ctx context.Context) (ios.ReadWriteCloser, string, error) {
		addr := hosts[r.Intn(len(hosts))]
		if !strings.Contains(addr, ":") {
			addr += ":7709"
		}
		c, err := net.Dial("tcp", addr)
		return c, addr, err
	}
}

func NewRangeDial(hosts []string) ios.DialFunc {
	return NewRangeDialWithTimeout(hosts, 0)
}

// NewRangeDialWithTimeout 遍历标准行情地址并限制单个地址的连接时间
func NewRangeDialWithTimeout(hosts []string, timeout time.Duration) ios.DialFunc {
	if len(hosts) == 0 {
		hosts = Hosts
	}
	dialer := &net.Dialer{Timeout: timeout}
	return func(ctx context.Context) (c ios.ReadWriteCloser, _ string, err error) {
		for i, addr := range hosts {
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			default:
			}
			if !strings.Contains(addr, ":") {
				addr += ":7709"
			}
			c, err = dialer.DialContext(ctx, "tcp", addr)
			if err == nil {
				return c, addr, nil
			}
			if i < len(hosts)-1 {
				//最后一个错误返回出去
				logs.Err(err, "等待2秒后尝试下一个服务地址...")
				if waitErr := waitContext(ctx, 2*time.Second); waitErr != nil {
					return nil, "", waitErr
				}
			}
		}
		return
	}
}

// NewExRangeDial 遍历扩展行情(TdxExHq)服务地址进行连接,成功则结束遍历(端口 7727)
func NewExRangeDial(hosts []string) ios.DialFunc {
	return NewExRangeDialWithTimeout(hosts, 0)
}

// NewExRangeDialWithTimeout 遍历扩展行情地址并限制单个地址的连接时间
func NewExRangeDialWithTimeout(hosts []string, timeout time.Duration) ios.DialFunc {
	if len(hosts) == 0 {
		hosts = ExHosts
	}
	dialer := &net.Dialer{Timeout: timeout}
	return func(ctx context.Context) (c ios.ReadWriteCloser, _ string, err error) {
		for i, addr := range hosts {
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			default:
			}
			if !strings.Contains(addr, ":") {
				addr += ":" + ExPort
			}
			c, err = dialer.DialContext(ctx, "tcp", addr)
			if err == nil {
				return c, addr, nil
			}
			if i < len(hosts)-1 {
				//最后一个错误返回出去
				logs.Err(err, "等待2秒后尝试下一个扩展行情服务地址...")
				if waitErr := waitContext(ctx, 2*time.Second); waitErr != nil {
					return nil, "", waitErr
				}
			}
		}
		return
	}
}

func waitContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
