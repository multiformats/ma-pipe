package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	ma "gx/ipfs/QmTYjPMCKGzhpfevCCu7j5rWDKRkVqQ1jusMM5HhyGEzD4/go-multiaddr"

	mapipe "github.com/jbenet/ma-pipe"
)

const VERSION = "1.0.0"

const USAGE = `USAGE
  ma-pipe <mode> <multiaddrs>...

  ma-pipe listen <listen-multiaddr1> <listen-multiaddr2>
  ma-pipe dial <dial-multiaddr1> <dial-multiaddr2>
  ma-pipe fwd <listen-multiaddr> <dial-multiaddr>
  ma-pipe proxy <listen-multiaddr>

OPTIONS
  -h, --help          display this help message
  -v, --version       display the version of the program
  -t, --trace <dir>   save a trace of the connection to <dir>

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

  # ma-pipe supports the /unix/stdio multiaddr
  ma-pipe fwd /unix/stdio /ip4/127.0.0.1/tcp/1234
`

type Opts struct {
	Mode    string
	Trace   string
	Version bool
	Addrs   []ma.Multiaddr
}

func parseArgs() (Opts, error) {

	// parse options
	o := Opts{Mode: "exit"}
	flag.BoolVar(&o.Version, "v", false, "")
	flag.BoolVar(&o.Version, "version", false, "")
	flag.StringVar(&o.Trace, "t", "", "")
	flag.StringVar(&o.Trace, "trace", "", "")
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

	return o, nil
}

func runMode(trace *mapipe.Trace, opts Opts) error {
	switch opts.Mode {
	case "listen":
		if len(opts.Addrs) != 2 {
			return errors.New("listen mode takes exactly 2 multiaddrs")
		} else {
			return mapipe.ListenPipe(opts.Addrs[0], opts.Addrs[1], trace)
		}
	case "dial":
		if len(opts.Addrs) != 2 {
			return errors.New("dial mode takes exactly 2 multiaddrs")
		} else {
			return mapipe.DialPipe(opts.Addrs[0], opts.Addrs[1], trace)
		}
	case "fwd":
		if len(opts.Addrs) != 2 {
			return errors.New("fwd mode takes exactly 2 multiaddrs")
		} else {
			return mapipe.ForwardPipe(opts.Addrs[0], opts.Addrs[1], trace)
		}
	case "proxy":
		if len(opts.Addrs) != 1 {
			return errors.New("proxy mode takes exactly 1 multiaddr")
		}
		return mapipe.ProxyPipe(opts.Addrs[0], trace)
	}
	return nil
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
