package eupho

import (
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"

	"github.com/soh335/sliceflag"
)

type Solo struct {
	FlagSet *flag.FlagSet

	Master  *Master
	timeout string

	Slave      *Slave
	jobs       string
	exec       string
	maxDelay   string
	maxRetry   string
	pluginArgs []string

	version bool

	wg sync.WaitGroup
}

func NewSolo() *Solo {
	s := &Solo{
		FlagSet: flag.NewFlagSet("eupho", flag.ExitOnError),
		Master:  NewMaster(),
		Slave:   NewSlave(),
	}

	// for master
	s.FlagSet.StringVar(&s.timeout, "timeout", "10m", "")

	// for slave
	s.FlagSet.StringVar(&s.jobs, "j", "1", "")
	s.FlagSet.StringVar(&s.jobs, "jobs", "1", "")
	s.FlagSet.StringVar(&s.exec, "exec", "perl", "")
	s.FlagSet.StringVar(&s.maxDelay, "max-delay", "3s", "")
	s.FlagSet.StringVar(&s.maxRetry, "max-retry", "10", "")
	sliceflag.StringVar(s.FlagSet, &s.pluginArgs, "plugin", []string{}, "plugins")
	sliceflag.StringVar(s.FlagSet, &s.pluginArgs, "P", []string{}, "plugins")

	s.FlagSet.BoolVar(&s.version, "version", false, "Show version of eupho-solo")

	return s
}

func (s *Solo) ParseArgs(args []string) {
	s.FlagSet.Parse(args)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	s.Master.ParseArgs([]string{
		"-timeout", s.timeout,
		"-addr", l.Addr().String(),
	})

	appendArgs := []string{}
	for _, p := range s.pluginArgs {
		appendArgs = append(appendArgs, "-plugin", p)
	}
	s.Slave.ParseArgs(append(
		[]string{
			"-addr", l.Addr().String(),
			"-jobs", s.jobs,
			"-exec", s.exec,
			"-max-delay", s.maxDelay,
			"-max-retry", s.maxRetry,
		},
		appendArgs...,
	))
}

func (s *Solo) Run(args []string) int {
	if args != nil {
		s.ParseArgs(args)
	}

	if s.version {
		fmt.Printf("eupho-solo %s, %s built for %s/%s\n", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return 0
	}

	code := 0
	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		code = s.Master.Run(nil)
	}()
	go func() {
		defer s.wg.Done()
		s.Slave.Run(nil)
	}()
	s.wg.Wait()

	return code
}
