package connection

import (
	"net"
)

type IntConsumer func(int)

type CountingConn struct {
	net.Conn
	readConsumer  IntConsumer
	writeConsumer IntConsumer
}

func (c *CountingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		c.readConsumer(n)
	}
	return n, err
}

func (c *CountingConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		c.writeConsumer(n)
	}
	return n, err
}

type CountingListener struct {
	net.Listener
	ReadConsumer  IntConsumer
	WriteConsumer IntConsumer
}

func (l *CountingListener) Accept() (net.Conn, error) {
	connection, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	cc := &CountingConn{Conn: connection, readConsumer: l.ReadConsumer, writeConsumer: l.WriteConsumer}
	return cc, nil
}
