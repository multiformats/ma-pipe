package mapipe

import (
	"errors"
	"fmt"
	"io"

	manet "gx/ipfs/QmP9wr7cpzyY76Tmpvcfe9vp2eMa1a3bVKA27wxPBBxez7/go-multiaddr-net"
	ma "gx/ipfs/QmTYjPMCKGzhpfevCCu7j5rWDKRkVqQ1jusMM5HhyGEzD4/go-multiaddr"
)

func pipeConn(a, b manet.Conn, t *Trace) error {
	if a == nil || b == nil {
		return errors.New("attempt to pipe nil manet.Conn")
	}

	fmt.Fprintln(t.CW, "piping", a.RemoteMultiaddr(), "to", b.RemoteMultiaddr())
	errs := make(chan error, 2)

	go func() {
		_, err := io.Copy(io.MultiWriter(a, t.AW), b)
		errs <- err
	}()

	go func() {
		_, err := io.Copy(io.MultiWriter(b, t.BW), a)
		errs <- err
	}()

	e1 := <-errs
	e2 := <-errs

	if e1 != nil {
		return e1
	}
	return e2
}

type ConnErr struct {
	Conn manet.Conn
	Err  error
}

// ListenPipe listens on both multiaddrs, accepts one connection each,
// and pipes them to each other.
func ListenPipe(l1, l2 ma.Multiaddr, t *Trace) error {

	list1, err := Listen(l1)
	if err != nil {
		return err
	}
	fmt.Fprintln(t.CW, "listening on", list1.Multiaddr())

	list2, err := Listen(l2)
	if err != nil {
		return err
	}
	fmt.Fprintln(t.CW, "listening on", list2.Multiaddr())

	conns := make(chan ConnErr, 2)
	acceptOneThenClose := func(l manet.Listener) {
		c, err := l.Accept()
		conns <- ConnErr{c, err}
		l.Close()
	}

	go acceptOneThenClose(list1)
	go acceptOneThenClose(list2)

	c1 := <-conns
	fmt.Fprintln(t.CW, "accepted", c1.Conn.LocalMultiaddr(), c1.Conn.RemoteMultiaddr())
	c2 := <-conns
	fmt.Fprintln(t.CW, "accepted", c2.Conn.LocalMultiaddr(), c2.Conn.RemoteMultiaddr())

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

	return pipeConn(c1.Conn, c2.Conn, t)
}

// ForwardPipe listens on one multiaddr, accepts one connection,
// dials to the second multiaddr, and pipes them to each other.
func ForwardPipe(l, d ma.Multiaddr, t *Trace) error {
	list, err := Listen(l)
	if err != nil {
		return err
	}
	fmt.Fprintln(t.CW, "listening on", list.Multiaddr())

	c1, err := list.Accept()
	list.Close()
	if err != nil {
		return err
	}
	defer c1.Close()
	fmt.Fprintln(t.CW, "accepted", c1.LocalMultiaddr(), c1.RemoteMultiaddr())

	fmt.Fprintln(t.CW, "dialing", d)
	c2, err := Dial(d)
	if err != nil {
		return err
	}
	defer c2.Close()
	fmt.Fprintln(t.CW, "dialed", c2.LocalMultiaddr(), c2.RemoteMultiaddr())

	return pipeConn(c1, c2, t)
}

// DialPipe dials to both multiaddrs, and pipes them to each other.
func DialPipe(d1, d2 ma.Multiaddr, t *Trace) error {
	fmt.Fprintln(t.CW, "dialing", d1)
	fmt.Fprintln(t.CW, "dialing", d2)

	c1, err := Dial(d1)
	if err != nil {
		return err
	}
	defer c1.Close()
	fmt.Fprintln(t.CW, "dialed", c1.LocalMultiaddr(), c1.RemoteMultiaddr())

	c2, err := Dial(d2)
	if err != nil {
		return err
	}
	defer c2.Close()
	fmt.Fprintln(t.CW, "dialed", c2.LocalMultiaddr(), c2.RemoteMultiaddr())

	return pipeConn(c1, c2, t)
}

// ProxyPipe listens on one multiaddr, reads a multiaddr, dials it, pipes them.
func ProxyPipe(l ma.Multiaddr, t *Trace) error {
	list, err := Listen(l)
	if err != nil {
		return err
	}
	fmt.Fprintln(t.CW, "listening on", list.Multiaddr())

	c1, err := list.Accept()
	list.Close()
	if err != nil {
		return err
	}
	defer c1.Close()
	fmt.Fprintln(t.CW, "accepted", c1.LocalMultiaddr(), c1.RemoteMultiaddr())

	// read until the first newline.
	d, err := readMultiaddr(c1)
	if err != nil {
		return err
	}
	fmt.Fprintln(t.CW, "requested proxy to", d)

	fmt.Fprintln(t.CW, "dialing", d)
	c2, err := Dial(d)
	if err != nil {
		return err
	}
	defer c2.Close()
	fmt.Fprintln(t.CW, "dialed", c2.LocalMultiaddr(), c2.RemoteMultiaddr())

	return pipeConn(c1, c2, t)
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
