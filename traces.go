package mapipe

import (
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Trace struct {
	CW io.Writer // for control messages
	AW io.Writer // a2b output
	BW io.Writer // b2a output
}

func OpenTraceFiles(t *Trace, dir string) error {
	// first, attempt mkdir -p.
	err := os.MkdirAll(dir, os.FileMode(0o755))
	if err != nil {
		return err
	}

	a, b, c := NewTraceFilenames()
	a = path.Join(dir, a)
	b = path.Join(dir, b)
	c = path.Join(dir, c)

	af, err := os.Create(a)
	if err != nil {
		return err
	}

	bf, err := os.Create(b)
	if err != nil {
		af.Close()
		return err
	}

	cf, err := os.Create(c)
	if err != nil {
		af.Close()
		bf.Close()
		return err
	}

	// wire up the trace files into the writers.
	t.AW = af
	t.BW = bf
	t.CW = io.MultiWriter(t.CW, cf)
	return nil
}

var (
	TraceFilenameFmt     = "ma-pipe-trace-<date>-<pid>-<direction>"
	TraceFilenameDateFmt = "2006-01-02-15:04:05Z"
)

func NewTraceFilenames() (string, string, string) {
	date := time.Now().Format(TraceFilenameDateFmt)
	s := strings.Replace(TraceFilenameFmt, "<date>", date, -1)
	s = strings.Replace(s, "<pid>", strconv.Itoa(os.Getpid()), -1)
	s1 := strings.Replace(s, "<direction>", "a2b", -1)
	s2 := strings.Replace(s, "<direction>", "b2a", -1)
	s3 := strings.Replace(s, "<direction>", "ctl", -1)
	return s1, s2, s3
}
