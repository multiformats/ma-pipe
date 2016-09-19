package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"io"
	"os"
	"os/signal"
	"syscall"
	cxt "context"

	ma "github.com/multiformats/go-multiaddr"
	mapipe "github.com/multiformats/ma-pipe"

	humanize "gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
)

const VERSION = "1.0.0"

var (
	ErrInvalidBandwidth = errors.New("Invalid bandwidth. Must of the form: 10MBps, 1Kbps, 1GB/s, ...")
)

const USAGE = `USAGE
	ma-pipe <mode> <multiaddrs>...

	ma-pipe listen <listen-multiaddr1> <listen-multiaddr2>
	ma-pipe dial <dial-multiaddr1> <dial-multiaddr2>
	ma-pipe fwd <listen-multiaddr> <dial-multiaddr>
	ma-pipe proxy <listen-multiaddr>

OPTIONS
	-h, --help               display this help message
	-v, --version            display the version of the program
	-t, --trace <dir>        save a trace of the connection to <dir>
	-e, --tee                tee the connection to stdio
	--bandwidth <bandwidth>  introduce a bandwidth cap (eg 1MB/s)

EXAMPLES
	# listen on two multiaddrs, accept 1 conn each, and pipe them
	ma-pipe listen /ip4/127.0.0.1/tcp/1234 /ip4/127.0.0.1/tcp/1234

	# dial to both multiaddrs, and pipe them
	ma-pipe dial /ip4/127.0.0.1/tcp/1234 /ip4/127.0.0.1/tcp/1234

	# listen on one multiaddr, accept 1 conn, dial to the other, and pipe them
	ma-pipe fwd /ip4/127.0.0.1/tcp/1234 /ip4/127.0.0.1/tcp/1234

	# listen on one multiaddr, accept 1 conn.
	# read the first line, parse a multiaddr, dial that multiaddr, and pipe them
	ma-pipe proxy /ip4/127.0.0.1/tcp/1234

	# ma-pipe supports "zero" listen multiaddrs
	ma-pipe proxy /ip4/0.0.0.0/tcp/0

	# ma-pipe supports the /unix/stdio multiaddr
	ma-pipe fwd /unix/stdio /ip4/127.0.0.1/tcp/1234

	# ma-pipe supports the --tee option to inspect conn in stdio
	ma-pipe --tee fwd /ip4/0.0.0.0/tcp/0 /ip4/127.0.0.1/tcp/1234

	# ma-pipe allows throttling connections with a bandwidth max
	ma-pipe --bandwidth 1MB/s listen /ip4/127.0.0.1/tcp/1234 /ip4/127.0.0.1/tcp/1234
`

type Opts struct {
	Mode      string
	Trace     string
	Version   bool
	Tee       bool
	BWidthStr string
	Bandwidth uint64 // in B/s
	Addrs     []ma.Multiaddr
}

func parseArgs() (Opts, error) {

	// parse options
	o := Opts{Mode: "exit"}
	flag.BoolVar(&o.Version, "v", false, "")
	flag.BoolVar(&o.Version, "version", false, "")
	flag.StringVar(&o.Trace, "t", "", "")
	flag.StringVar(&o.Trace, "trace", "", "")
	flag.BoolVar(&o.Tee, "e", false, "")
	flag.BoolVar(&o.Tee, "tee", false, "")
	flag.StringVar(&o.BWidthStr, "bandwidth", "", "")
	flag.Usage = func() {
		fmt.Print(USAGE)
	}
	flag.Parse()

	if o.Version {
		fmt.Println("ma-pipe", VERSION)
		return o, nil
	}

	args := flag.Args()
	if len(args) < 2 { // <mode> <addrs>+
		fmt.Print(USAGE)
		return o, errors.New("not enough arguments")
	}

	// set the mode
	o.Mode = args[0]

	// parse the multiaddrs
	o.Addrs = make([]ma.Multiaddr, len(args)-1)
	for i, saddr := range args[1:] {
		maddr, err := ma.NewMultiaddr(saddr)
		if err != nil {
			return o, err
		}
		o.Addrs[i] = maddr
	}

	// parse the bandwidth
	var err error
	o.Bandwidth, err = ParseBandwidth(o.BWidthStr)
	if err != nil {
		return o, err
	}

	return o, nil
}

// ParseBandwidth parses a bandwidth string of the form: <size>/s (eg 10MB/s, 1KBps)
func ParseBandwidth(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}
	if len(s) < 4 {
		return 0, ErrInvalidBandwidth
	}
	if s[len(s)-1] != 's' {
		return 0, ErrInvalidBandwidth
	}
	if s[len(s)-2] != '/' && s[len(s)-2] != 'p' {
		return 0, ErrInvalidBandwidth
	}
	return humanize.ParseBytes(s[0:len(s)-2])
}

func catchSigPipe(ctx cxt.Context, trace *mapipe.Trace) cxt.Context {
	ctx2, cancel := cxt.WithCancel(cxt.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGPIPE)
		<-c
		fmt.Fprintln(trace.CW, "received SIGPIPE, closing...")
		cancel()
	}()
	return ctx2
}

func runMode(trace *mapipe.Trace, opts Opts) error {
	o := mapipe.Opts{trace, opts.Bandwidth, nil}

	// setup the kill channel for the pipe. This kill channel solves the following problem:
	// when SIGPIPE is sent, the output cuts and we do not have a graceful exit. This enables
	// nicer printing when using ma-pipe with other programs.
	ctx := catchSigPipe(cxt.Background(), trace)

	switch opts.Mode {
	case "listen":
		if len(opts.Addrs) != 2 {
			return errors.New("listen mode takes exactly 2 multiaddrs")
		}
		return mapipe.ListenPipe(ctx, opts.Addrs[0], opts.Addrs[1], o)
	case "dial":
		if len(opts.Addrs) != 2 {
			return errors.New("dial mode takes exactly 2 multiaddrs")
		}
		return mapipe.DialPipe(ctx, opts.Addrs[0], opts.Addrs[1], o)
	case "fwd":
		if len(opts.Addrs) != 2 {
			return errors.New("fwd mode takes exactly 2 multiaddrs")
		}
		return mapipe.ForwardPipe(ctx, opts.Addrs[0], opts.Addrs[1], o)
	case "proxy":
		if len(opts.Addrs) != 1 {
			return errors.New("proxy mode takes exactly 1 multiaddr")
		}
		return mapipe.ProxyPipe(ctx, opts.Addrs[0], o)
	}
	return fmt.Errorf("invalid mode %s", opts.Mode)
}

func run() error {
	opts, err := parseArgs()
	if err != nil {
		return nil
	}

	switch opts.Mode {
	case "listen", "dial", "fwd", "proxy":

		trace := &mapipe.Trace{
			CW: os.Stderr,
			AW: ioutil.Discard,
			BW: ioutil.Discard,
		}

		if opts.Tee {
			trace.CW = NewPrefixWriter(os.Stderr, "# ")
			trace.AW = NewPrefixWriter(os.Stdout, "> ")
			trace.BW = NewPrefixWriter(os.Stdout, "< ")
		}

		if opts.Trace != "" {
			err := mapipe.OpenTraceFiles(trace, opts.Trace)
			if err != nil {
				return err
			}
		}

		err := runMode(trace, opts)
		if err != nil {
			fmt.Fprintf(trace.CW, "error: %s\n", err)
		}
		return err

	case "exit":
		return nil

	default:
		fmt.Print(USAGE)
		return fmt.Errorf("invalid mode %s", opts.Mode)
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}


type PrefixWriter struct {
	W      io.Writer
	Prefix []byte
}

func NewPrefixWriter(W io.Writer, pre string) *PrefixWriter {
	return &PrefixWriter{W, []byte(pre)}
}

func (pw *PrefixWriter) Write(buf []byte) (int, error) {
	buf = append(pw.Prefix, buf...)
	n, err := pw.W.Write(buf)

	// have to remove the length
	n = n - len(pw.Prefix)
	if n < 0 {
		n = 0
	}

	return n, err
}

func (pw *PrefixWriter) Close() (error) {
	if c, ok := pw.W.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
