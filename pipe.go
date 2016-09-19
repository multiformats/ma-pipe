package mapipe

import (
  "errors"
  "fmt"
  "time"
  "io"
  cxt "context"

  ctxio "github.com/jbenet/go-context/io"
  manet "github.com/multiformats/go-multiaddr-net"
  ma "github.com/multiformats/go-multiaddr"
  humanize "gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
)

type xmitResult struct {
  n    int64
  err  error
  from ma.Multiaddr
  to   ma.Multiaddr
}

func pipeConn(ctx1 cxt.Context, a, b manet.Conn, o Opts) error {
  ctx, cancel := cxt.WithCancel(ctx1)
  if a == nil || b == nil {
    return errors.New("attempt to pipe nil manet.Conn")
  }

  fmt.Fprintln(o.Trace.CW, "piping", a.RemoteMultiaddr(), "<-->", b.RemoteMultiaddr())
  if o.MaxBW > 0 {
    fmt.Fprintf(o.Trace.CW, "rate-limiting to %v/s\n", humanize.Bytes(o.MaxBW))
  }

  xmits := make(chan xmitResult, 2)
  xmitRLC := func(c1, c2 manet.Conn) {
    w := ctxio.NewWriter(ctx, io.MultiWriter(c1, o.Trace.AW))
    r := ctxio.NewReader(ctx, c2)
    n, err := rateLimitedCopy(w, r, o)
    xmits <- xmitResult{n, err, c2.RemoteMultiaddr(), c1.RemoteMultiaddr()}
  }
  go xmitRLC(a, b)
  go xmitRLC(b, a)
  go func() {
    <-ctx.Done()
    a.Close()
    b.Close()
  }()

  var err error
  ignoreErr := func(e error) bool {
    return e == io.EOF ||
          e == io.ErrUnexpectedEOF ||
          e == cxt.Canceled ||
          e == cxt.DeadlineExceeded
  }

  printResults := func(x xmitResult) {
    fmt.Fprintln(o.Trace.CW, "wrote", x.n, "bytes from", x.from, "to", x.to)
    if x.err != nil && !ignoreErr(x.err) {
      fmt.Fprintln(o.Trace.CW, x.from, "connection failed:", x.err)
      err = x.err
    }
  }

  printResults(<-xmits)
  cancel() // stop when any side closes. this is same as nc behavior
  printResults(<-xmits)
  return nil
}

func rateLimitedCopy(dst io.Writer, src io.Reader, o Opts) (written int64, err error) {
  if o.MaxBW < 1 { // unlimited
    return io.Copy(dst, src)
  }
  buf := make([]byte, o.MaxBW)

  tstart := time.Now() // time now
  telapsed := time.Duration(0)
  texpected := time.Duration(0)
  for {
    nr, er := src.Read(buf)
    if nr > 0 {
      nw, ew := dst.Write(buf[0:nr])
      if nw > 0 {
        written += int64(nw)
      }
      if ew != nil {
        err = ew
        break
      }
      if nr != nw {
        err = io.ErrShortWrite
        break
      }
    }
    if er == io.EOF {
      break
    }
    if er != nil {
      err = er
      break
    }

    telapsed = time.Since(tstart)
    texpected = time.Second * time.Duration(written / int64(o.MaxBW))
    if texpected > telapsed { // roughly how many seconds we should've waited
      time.Sleep(texpected - telapsed)
    }
  }
  return written, err
}
