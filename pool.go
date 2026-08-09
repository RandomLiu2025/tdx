package tdx

import (
	"errors"

	"github.com/injoyai/base/safe"
)

type (
	IPool interface {
		Get() (*Client, error)
		Put(c *Client)
		Do(fn func(c *Client) error) error
		Go(fn func(c *Client)) error
	}
	DialPoolFunc = func() (IPool, error)
)

// NewPool 简易版本的连接池
func NewPool(dial func() (*Client, error), number int) (*Pool, error) {
	if number <= 0 {
		number = 1
	}
	ch := make(chan *Client, number)
	p := &Pool{
		ch: ch,
		Closer: safe.NewCloser().SetCloseFunc(func(err error) error {
			close(ch)
			return nil
		}),
	}
	for i := 0; i < number; i++ {
		c, err := dial()
		if err != nil {
			for _, connectedClient := range p.clients {
				if connectedClient != nil && connectedClient.Client != nil {
					_ = connectedClient.CloseAll()
				}
			}
			return nil, err
		}
		p.clients = append(p.clients, c)
		p.ch <- c
	}
	return p, nil
}

type Pool struct {
	ch      chan *Client
	clients []*Client
	*safe.Closer
}

// Ready 是否至少有一个已连接客户端
func (this *Pool) Ready() bool {
	if this == nil {
		return false
	}
	for _, client := range this.clients {
		if client.Connected() {
			return true
		}
	}
	return false
}

func (this *Pool) Get() (*Client, error) {
	select {
	case <-this.Done():
		return nil, this.Err()
	case c, ok := <-this.ch:
		if !ok {
			return nil, errors.New("已关闭")
		}
		return c, nil
	}
}

func (this *Pool) Put(c *Client) {
	select {
	case <-this.Done():
		c.Close()
		return
	case this.ch <- c:
	}
}

func (this *Pool) Do(fn func(c *Client) error) error {
	c, err := this.Get()
	if err != nil {
		return err
	}
	defer this.Put(c)
	return fn(c)
}

func (this *Pool) Go(fn func(c *Client)) error {
	c, err := this.Get()
	if err != nil {
		return err
	}
	go func(c *Client) {
		defer this.Put(c)
		fn(c)
	}(c)
	return nil
}
