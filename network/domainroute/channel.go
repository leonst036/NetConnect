package domainroute

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Channel struct {
	ID        uint32
	conn      net.Conn
	inCh      chan []byte
	closed    atomic.Bool
	closeOnce sync.Once
	done      chan struct{}
}

func NewChannel(id uint32, conn net.Conn) *Channel {
	return &Channel{
		ID:   id,
		conn: conn,
		inCh: make(chan []byte, 128),
		done: make(chan struct{}),
	}
}

func (c *Channel) PushIncoming(data []byte) bool {
	if c.closed.Load() {
		return false
	}

	copied := make([]byte, len(data))
	copy(copied, data)

	select {
	case c.inCh <- copied:
		return true
	case <-c.done:
		return false
	case <-time.After(5 * time.Second):
		c.Close()
		return false
	}
}

func (c *Channel) RunEgress() {
	for {
		select {
		case data, ok := <-c.inCh:
			if !ok {
				return
			}
			_, err := c.conn.Write(data)
			if err != nil {
				c.Close()
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *Channel) Close() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		close(c.done)
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
}

func (c *Channel) IsClosed() bool {
	return c.closed.Load()
}

func (c *Channel) Conn() net.Conn {
	return c.conn
}
