# ma-pipe - multiaddr pipes

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-multiformats-blue.svg?style=flat-square)](http://github.com/multiformats/multiformats)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)

> multiaddr powered pipes

This is a simple program, much like netcat or telnet, that sets up pipes between [multiaddrs](https://github.com/multiformats/multiaddr).

## Table of Contents

- [Install](#install)
- [Usage](#usage)
  - [CLI Usage Text](#cli-usage-text)
  - [Tee (`--tee`)](#tee---tee)
  - [Traces (`--trace`)](#traces---trace)
- [Maintainer](#maintainer)
- [Contribute](#contribute)
- [License](#license)

## Install

For now, use `go get` to install it:

```
go get -u github.com/multiformats/ma-pipe/ma-pipe
ma-pipe --version # should work
```

Please file an issue if there is a problem with the install.

## Usage

`ma-pipe` sets up simple pipes based on multiaddrs. It has four modes:

- `listen` will listen on 2 given multiaddrs, accept 1 conn each, and pipe the connection together
- `dial` will dial to 2 given multiaddrs, and pipe the connection together
- `fwd` will listen on 1 multiaddr, accept 1 conn, then dial the other given multiaddr
- `proxy` will listen on 1 multiaddr, accept 1 conn, read a multiaddr from the conn, and dial it

Notes:

- `ma-pipe` supports "zero" listen multiaddrs (eg `ma-pipe proxy /ip4/0.0.0.0/tcp/0`)
- `ma-pipe` supports the `/unix/stdio` multiaddr (eg `ma-pipe fwd /unix/stdio /ip4/127.0.0.1/tcp/1234`)

### CLI Usage Text

```sh
> ma-pipe --help
USAGE
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

	# ma-pipe supports "zero" listen multiaddrs
	ma-pipe proxy /ip4/0.0.0.0/tcp/0

	# ma-pipe supports the /unix/stdio multiaddr
	ma-pipe fwd /unix/stdio /ip4/127.0.0.1/tcp/1234

	# ma-pipe supports the --tee option to inspect conn in stdio
	ma-pipe --tee fwd /ip4/0.0.0.0/tcp/0 /ip4/127.0.0.1/tcp/1234
```

### Tee (`--tee`)

The `-e, --tee` option allows the user to inspect the connection in stdio, as it happens.

```sh
> ma-pipe --tee listen /ip4/127.0.0.1/tcp/64829 /ip4/127.0.0.1/tcp/64830
# listening on /ip4/127.0.0.1/tcp/64829
# listening on /ip4/127.0.0.1/tcp/64830
# accepted /ip4/127.0.0.1/tcp/64830 /ip4/127.0.0.1/tcp/64853
# accepted /ip4/127.0.0.1/tcp/64829 /ip4/127.0.0.1/tcp/64855
# piping /ip4/127.0.0.1/tcp/64853 to /ip4/127.0.0.1/tcp/64855
> Hello there
< Hi!
> How's it going?
< Well, and you?
```


### Traces (`--trace`)

The `-t, --trace` option allows the user to specify a directory to capture a trace of the connection. Three files will be written:

- `<trace-dir>/ma-pipe-trace-<date>-<pid>-a2b` for one side of the (duplex) connection.
- `<trace-dir>/ma-pipe-trace-<date>-<pid>-b2a` for the other side of the (duplex) connection.
- `<trace-dir>/ma-pipe-trace-<date>-<pid>-ctl` for control messages.

```
> tree mytraces
mytraces
├── ma-pipe-trace-2016-09-12-03:35:31Z-14088-a2b
├── ma-pipe-trace-2016-09-12-03:35:31Z-14088-b2a
└── ma-pipe-trace-2016-09-12-03:35:31Z-14088-ctl
```

## Maintainer

Captain: [@jbenet](https://github.com/jbenet).

## Contribute

Contributions welcome. Please check out [the issues](https://github.com/multiformats/ma-pipe/issues).

Check out our [contributing document](https://github.com/multiformats/multiformats/blob/master/contributing.md) for more information on how we work, and about contributing in general. Please be aware that all interactions related to multiformats are subject to the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

Small note: If editing the Readme, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

[MIT](LICENSE)
