package eupho

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"sync"

	"github.com/jessevdk/go-flags"
)

type Solo struct {
	Master *Master
	Slave  *Slave

	wg sync.WaitGroup

	opts soloOptions
}

type soloOptions struct {
	Jobs       string   `short:"j" long:"jobs"      default:"1"    description:"Run N test jobs in parallel"`
	Exec       string   `          long:"exec"      default:"perl" description:""`
	Merge      bool     `          long:"merge"                    description:"Merge test scripts' STDERR with their STDOUT"`
	PluginArgs []string `short:"P" long:"plugin"                   description:"plugins"`
	Version    bool     `          long:"version"                  description:"Show version of eupho-slave"`
	MaxDelay   string   `          long:"max-delay" default:"3s"   description:"Max delay duration"`
	MaxRetry   string   `          long:"max-retry" default:"10"   description:"Max retry num"`
	Timeout    string   `          long:"timeout"   default:"10m"  description:"Timeout duration"`
	Quiet      bool     `short:"q" long:"quiet"                    description:"quiet"`
}

func NewSolo() *Solo {
	return &Solo{
		Master: NewMaster(),
		Slave:  NewSlave(),
	}
}

func (s *Solo) ParseArgs(args []string) {
	var opts soloOptions
	parser := flags.NewParser(
		&opts,
		flags.HelpFlag|flags.PassDoubleDash,
	)
	moreArgs, err := parser.ParseArgs(args)
	if err != nil {
		fmt.Println(err)
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}
	s.opts = opts

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	s.Master.ParseArgs([]string{
		"--timeout", s.opts.Timeout,
		"--addr", l.Addr().String(),
		"--quiet",
	})

	slaveArgs := []string{
		"--addr", l.Addr().String(),
		"--jobs", s.opts.Jobs,
		"--exec", s.opts.Exec,
		"--max-delay", s.opts.MaxDelay,
		"--max-retry", s.opts.MaxRetry,
	}
	for _, p := range s.opts.PluginArgs {
		slaveArgs = append(slaveArgs, "--plugin", p)
	}
	if s.opts.Quiet {
		slaveArgs = append(slaveArgs, "--quiet")
	}
	if s.opts.Merge {
		slaveArgs = append(slaveArgs, "--merge")
	}
	s.Slave.ParseArgs(append(slaveArgs, moreArgs...))
}

func (s *Solo) Run(args []string) int {
	if args != nil {
		s.ParseArgs(args)
	}

	if s.opts.Version {
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
