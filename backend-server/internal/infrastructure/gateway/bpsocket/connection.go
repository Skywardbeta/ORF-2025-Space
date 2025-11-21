// connection.go - 再接続機能付きBPソケット接続の管理
package bpsocket

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Connection struct {
	socket           *BpSocket
	localNodeNum     uint64
	localSvcNum      uint64
	remoteNodeNum    uint64
	remoteSvcNum     uint64
	reconnectBackoff time.Duration
	mu               sync.RWMutex
	closed           bool
}

func NewConnection(localNodeNum, localSvcNum, remoteNodeNum, remoteSvcNum uint64) (*Connection, error) {
	socket, err := NewBpSocket(localNodeNum, localSvcNum)
	if err != nil {
		return nil, err
	}

	return &Connection{
		socket:           socket,
		localNodeNum:     localNodeNum,
		localSvcNum:      localSvcNum,
		remoteNodeNum:    remoteNodeNum,
		remoteSvcNum:     remoteSvcNum,
		reconnectBackoff: 1 * time.Second,
	}, nil
}

func (c *Connection) Send(ctx context.Context, data []byte) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("connection closed")
	}
	socket := c.socket
	c.mu.RUnlock()

	err := socket.Send(data, c.remoteNodeNum, c.remoteSvcNum)
	if err != nil {
		log.Printf("[BpSocket] Send failed, reconnecting: %v", err)
		if reconnectErr := c.reconnect(ctx); reconnectErr != nil {
			return fmt.Errorf("send failed: %w", err)
		}
		return c.socket.Send(data, c.remoteNodeNum, c.remoteSvcNum)
	}
	return nil
}

func (c *Connection) Recv(buf []byte) (int, *SockaddrBP, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return 0, nil, fmt.Errorf("connection closed")
	}
	socket := c.socket
	c.mu.RUnlock()

	n, addr, err := socket.Recv(buf)
	if err != nil {
		log.Printf("[BpSocket] Recv failed, may need reconnect: %v", err)
	}
	return n, addr, err
}

func (c *Connection) Reconnect(ctx context.Context) error {
	return c.reconnect(ctx)
}

func (c *Connection) reconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.socket != nil {
		_ = c.socket.Close()
	}

	backoff := c.reconnectBackoff
	for attempt := 1; attempt <= 3; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		log.Printf("[BpSocket] Reconnect attempt %d/3", attempt)
		socket, err := NewBpSocket(c.localNodeNum, c.localSvcNum)
		if err == nil {
			c.socket = socket
			log.Printf("[BpSocket] Reconnected: %s", socket.LocalAddr().String())
			return nil
		}

		log.Printf("[BpSocket] Reconnect failed: %v, retry in %v", err, backoff)
		time.Sleep(backoff)
		backoff *= 2
	}

	return fmt.Errorf("max retry exceeded")
}

func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	if c.socket != nil {
		return c.socket.Close()
	}
	return nil
}

func (c *Connection) LocalAddr() *SockaddrBP {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.socket != nil {
		return c.socket.LocalAddr()
	}
	return nil
}
