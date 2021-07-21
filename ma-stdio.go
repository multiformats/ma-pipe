package mapipe

import (
	"errors"
	"io"
	"net"
	"os"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

var (
	StdioMultiaddr    = ma.StringCast("/unix/stdio")
	ProcStdioListener = StdioListener{}
	ProcStdioConn     = IOConn{
		R:     os.Stdin,
		W:     os.Stdout,
		LAddr: StdioMultiaddr,
		RAddr: StdioMultiaddr,
	}
)

func Listen(a ma.Multiaddr) (manet.Listener, error) {
	if StdioMultiaddr.Equal(a) {
		return &ProcStdioListener, nil
	}

	return manet.Listen(a)
}

func Dial(a ma.Multiaddr) (manet.Conn, error) {
	if StdioMultiaddr.Equal(a) {
		return &ProcStdioConn, nil
	}

	return manet.Dial(a)
}

type StdioListener struct {
	accepted bool // can only accept once
	conn     IOConn
}

func (sl *StdioListener) NetListener() net.Listener {
	return nil
}

func (sl *StdioListener) Accept() (manet.Conn, error) {
	if sl.accepted {
		return nil, errors.New("no more connections")
	}

	sl.conn = ProcStdioConn
	sl.accepted = true
	return &sl.conn, nil
}

func (sl *StdioListener) Close() error {
	return nil
}

func (sl *StdioListener) Multiaddr() ma.Multiaddr {
	return StdioMultiaddr
}

func (sl *StdioListener) Addr() net.Addr {
	return nil
}

type IOConn struct {
	R     io.Reader
	W     io.Writer
	LAddr ma.Multiaddr
	RAddr ma.Multiaddr
}

func (c *IOConn) Read(b []byte) (n int, err error) {
	return c.R.Read(b)
}

func (c *IOConn) Write(b []byte) (n int, err error) {
	return c.W.Write(b)
}

func (c *IOConn) Close() (err error) {
	if rc, ok := c.R.(io.Closer); ok {
		err = rc.Close()
	}
	if wc, ok := c.W.(io.Closer); ok {
		err = wc.Close()
	}
	return
}

func (c *IOConn) LocalAddr() net.Addr                { return nil }
func (c *IOConn) RemoteAddr() net.Addr               { return nil }
func (c *IOConn) SetDeadline(t time.Time) error      { return nil }
func (c *IOConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *IOConn) SetWriteDeadline(t time.Time) error { return nil }
func (c *IOConn) LocalMultiaddr() ma.Multiaddr       { return c.LAddr }
func (c *IOConn) RemoteMultiaddr() ma.Multiaddr      { return c.RAddr }
