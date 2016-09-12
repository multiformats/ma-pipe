#!/bin/bash

# config
DIR="tmp"
PIPE="./ma-pipe"
BIN_1="./ipfs"
BIN_2="./ipfs"

# logging setup
LOG="$DIR/log"
log() {
  echo "# $@" >>"$LOG"
  echo "# $@"
}

run() {
  echo "> $@" >>"$LOG"
  echo "> $@"
  eval "$@"
}

ipfs1() {
  run IPFS_PATH="$PATH_1" "$BIN_1" "$@"
}

ipfs2() {
  run IPFS_PATH="$PATH_2" "$BIN_2" "$@"
}

die() {
  echo >&2 "$@"
  exit 1
}

# checks first
type -t "$PIPE" >/dev/null || die "error: no ma-pipe. please run make"
type -t "$BIN_1" >/dev/null || die "error: no $BIN_2. please correct BIN_1 path."
type -t "$BIN_2" >/dev/null || die "error: no $BIN_2. please correct BIN_2 path."

# starting...
mkdir -p "$DIR"
echo "# $0 run at $(date)" >"$LOG"
log "logging to $LOG"

PATH_1="$DIR/repo1"
PATH_2="$DIR/repo2"
log "using $PIPE ($($PIPE --version))"
log "using $BIN_1 ($($BIN_1 version)) $PATH_1"
log "using $BIN_2 ($($BIN_2 version)) $PATH_2"

log "init both nodes"
mkdir -p "$PATH_1"
mkdir -p "$PATH_2"
ipfs1 init >>"$LOG" 2>>"$LOG"
ipfs2 init >>"$LOG" 2>>"$LOG"

log "get peer ids"
run ID1=$(IPFS_PATH="$PATH_1" "$BIN_1" id -f "<id>")
run ID2=$(IPFS_PATH="$PATH_2" "$BIN_2" id -f "<id>")

log "clear bootstrap list"
ipfs1 bootstrap rm --all
ipfs2 bootstrap rm --all

log "set swarm addrs"
ADDR1=/ip4/127.0.0.1/tcp/4101
ADDR2=/ip4/127.0.0.1/tcp/4102
ADDR3=/ip4/127.0.0.1/tcp/4103
ipfs1 config --json Addresses.Swarm "'[\"$ADDR1\"]'"
ipfs2 config --json Addresses.Swarm "'[\"$ADDR2\"]'"
ipfs1 config Addresses.API "/ip4/127.0.0.1/tcp/5101"
ipfs2 config Addresses.API "/ip4/127.0.0.1/tcp/5102"
ipfs1 config Addresses.Gateway "/ip4/127.0.0.1/tcp/5201"
ipfs2 config Addresses.Gateway "/ip4/127.0.0.1/tcp/5202"
log "peer1 Addresses $(IPFS_PATH="$PATH_1" "$BIN_1" config Addresses)"
log "peer2 Addresses $(IPFS_PATH="$PATH_2" "$BIN_2" config Addresses)"

log "launch daemons without transport encryption"
ipfs1 "daemon --disable-transport-encryption &"
PID1=$!
ipfs2 "daemon --disable-transport-encryption &"
PID2=$!

log "wait for them to come up..."
sleep 5

log "setup a fwd pipe with trace"
run "$PIPE -t $DIR/traces fwd $ADDR3 $ADDR2 &"
PID3=$!

log "connect peer1 to pipe, which will fwd to peer2"
ipfs1 swarm connect "$ADDR3/ipfs/$ID2"

log "now add some files"
ipfs1 add "$0"
ipfs2 add "$0"

log "let them do stuff for 10 seconds (sleep)..."
sleep 10

log "kill them and exit"
run kill -9 "$PID3"
run kill -9 "$PID1"
run kill -9 "$PID2"
