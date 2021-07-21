package mapipe

import (
	cxt "context"
	"errors"
	"fmt"
	"io"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

type Opts struct {
	Trace *Trace // trace object
	MaxBW uint64 // in Bytes/s
	Kill  <-chan struct{}
}

type ConnErr struct {
	Conn manet.Conn
	Err  error
}

// ListenPipe listens on both multiaddrs, accepts one connection each,
// and pipes them to each other.
func ListenPipe(ctx cxt.Context, l1, l2 ma.Multiaddr, o Opts) error {
	list1, err := Listen(l1)
	if err != nil {
		return err
	}
	fmt.Fprintln(o.Trace.CW, "listening on", list1.Multiaddr())

	list2, err := Listen(l2)
	if err != nil {
		return err
	}
	fmt.Fprintln(o.Trace.CW, "listening on", list2.Multiaddr())

	conns := make(chan ConnErr, 2)
	acceptOneThenClose := func(l manet.Listener) {
		c, err := l.Accept()
		conns <- ConnErr{c, err}
		l.Close()
	}

	go acceptOneThenClose(list1)
	go acceptOneThenClose(list2)

	c1 := <-conns
	fmt.Fprintln(o.Trace.CW, "accepted", c1.Conn.LocalMultiaddr(), c1.Conn.RemoteMultiaddr())
	c2 := <-conns
	fmt.Fprintln(o.Trace.CW, "accepted", c2.Conn.LocalMultiaddr(), c2.Conn.RemoteMultiaddr())

	defer func() {
		if c1.Conn != nil {
			c1.Conn.Close()
		}
		if c2.Conn != nil {
			c2.Conn.Close()
		}
	}()

	if c1.Err != nil {
		return c1.Err
	}

	if c2.Err != nil {
		return c2.Err
	}

	return pipeConn(ctx, c1.Conn, c2.Conn, o)
}

// ForwardPipe listens on one multiaddr, accepts one connection,
// dials to the second multiaddr, and pipes them to each other.
func ForwardPipe(ctx cxt.Context, l, d ma.Multiaddr, o Opts) error {
	list, err := Listen(l)
	if err != nil {
		return err
	}
	fmt.Fprintln(o.Trace.CW, "listening on", list.Multiaddr())

	c1, err := list.Accept()
	list.Close()
	if err != nil {
		return err
	}
	defer c1.Close()
	fmt.Fprintln(o.Trace.CW, "accepted", c1.LocalMultiaddr(), c1.RemoteMultiaddr())

	fmt.Fprintln(o.Trace.CW, "dialing", d)
	c2, err := Dial(d)
	if err != nil {
		return err
	}
	defer c2.Close()
	fmt.Fprintln(o.Trace.CW, "dialed", c2.LocalMultiaddr(), c2.RemoteMultiaddr())

	return pipeConn(ctx, c1, c2, o)
}

// DialPipe dials to both multiaddrs, and pipes them to each other.
func DialPipe(ctx cxt.Context, d1, d2 ma.Multiaddr, o Opts) error {
	fmt.Fprintln(o.Trace.CW, "dialing", d1)
	fmt.Fprintln(o.Trace.CW, "dialing", d2)

	c1, err := Dial(d1)
	if err != nil {
		return err
	}
	defer c1.Close()
	fmt.Fprintln(o.Trace.CW, "dialed", c1.LocalMultiaddr(), c1.RemoteMultiaddr())

	c2, err := Dial(d2)
	if err != nil {
		return err
	}
	defer c2.Close()
	fmt.Fprintln(o.Trace.CW, "dialed", c2.LocalMultiaddr(), c2.RemoteMultiaddr())

	return pipeConn(ctx, c1, c2, o)
}

// ProxyPipe listens on one multiaddr, reads a multiaddr, dials it, pipes them.
func ProxyPipe(ctx cxt.Context, l ma.Multiaddr, o Opts) error {
	list, err := Listen(l)
	if err != nil {
		return err
	}
	fmt.Fprintln(o.Trace.CW, "listening on", list.Multiaddr())

	c1, err := list.Accept()
	list.Close()
	if err != nil {
		return err
	}
	defer c1.Close()
	fmt.Fprintln(o.Trace.CW, "accepted", c1.LocalMultiaddr(), c1.RemoteMultiaddr())

	// read until the first newline.
	d, err := readMultiaddr(c1)
	if err != nil {
		return err
	}
	fmt.Fprintln(o.Trace.CW, "requested proxy to", d)

	fmt.Fprintln(o.Trace.CW, "dialing", d)
	c2, err := Dial(d)
	if err != nil {
		return err
	}
	defer c2.Close()
	fmt.Fprintln(o.Trace.CW, "dialed", c2.LocalMultiaddr(), c2.RemoteMultiaddr())

	return pipeConn(ctx, c1, c2, o)
}

func readMultiaddr(r io.Reader) (ma.Multiaddr, error) {
	buf := make([]byte, 2048)
	for i := 0; i < len(buf); i++ {
		_, err := r.Read(buf[i : i+1])
		if err != nil {
			return nil, err
		}

		if buf[i] == byte('\n') {
			// found the newline
			return ma.NewMultiaddr(string(buf[0:i]))
		}
	}

	return nil, errors.New("did not find expected multiaddr and newline")
}
