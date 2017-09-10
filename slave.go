package eupho

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Songmu/retry"
	"github.com/jessevdk/go-flags"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Slave struct {
	Merge   bool
	Plugins []Plugin

	chanTests  chan *Test
	chanSuites chan *Test
	wgWorkers  *sync.WaitGroup

	opts slaveOptions
	args []string

	submitted bool
}

type slaveOptions struct {
	Addr       string        `             long:"addr"      default:"127.0.0.1:19300" description:"Listen addr"`
	Jobs       int           `short:"j"    long:"jobs"                                description:"Run N test jobs in parallel"`
	Exec       string        `             long:"exec"      default:"perl"            description:""`
	Merge      bool          `             long:"merge"                               description:"Merge test scripts' STDERR with their STDOUT"`
	PluginArgs []string      `short:"P"    long:"plugin"                              description:"plugins"`
	Version    bool          `             long:"version"                             description:"Show version of eupho-slave"`
	MaxDelay   time.Duration `             long:"max-delay" default:"3s"              description:"Max delay duration"`
	MaxRetry   uint          `             long:"max-retry" default:"10"              description:"Max retry num"`
	Quiet      bool          `short:"q"    long:"quiet"                               description:"quiet"`
}

func NewSlave() *Slave {
	return &Slave{
		Plugins:    []Plugin{},
		chanTests:  make(chan *Test),
		chanSuites: make(chan *Test),
		wgWorkers:  &sync.WaitGroup{},
	}
}

func (s *Slave) ParseArgs(args []string) {
	var (
		opts slaveOptions
		err  error
	)
	parser := flags.NewParser(
		&opts,
		flags.HelpFlag|flags.PassDoubleDash,
	)
	s.args, err = parser.ParseArgs(args)
	if err != nil {
		fmt.Println(err)
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	s.opts = opts
	if s.opts.Jobs < 1 {
		s.opts.Jobs = 1
	}

	for _, plugin := range s.opts.PluginArgs {
		a := strings.SplitN(plugin, "=", 2)
		name := a[0]
		pluginArgs := ""
		if len(a) >= 2 {
			pluginArgs = a[1]
		}

		loader, ok := pluginLoaders[name]
		if !ok {
			panic("plugin " + name + " not found")
		}
		s.Plugins = append(s.Plugins, loader.Load(name, pluginArgs))
	}
}

func (s *Slave) Run(args []string) {
	if args != nil {
		s.ParseArgs(args)
	}

	if s.opts.Version {
		fmt.Printf("eupho-slave %s, %s built for %s/%s\n", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return
	}

	for i := 0; i < s.opts.Jobs; i++ {
		w := NewWorker(s)
		w.Start()
	}

	testFiles := s.findTestFiles()

	conn, err := grpc.Dial(
		s.opts.Addr,
		grpc.WithInsecure(),
		grpc.WithBackoffMaxDelay(s.opts.MaxDelay),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := NewEuphoClient(conn)

	go func() {
		for {
			var path string
			err := retry.Retry(s.opts.MaxRetry, s.opts.MaxDelay, func() error {
				req := &GetTestRequest{Submitted: s.submitted}
				if !s.submitted {
					req.TestFiles = testFiles
				}
				res, err := client.GetTest(context.Background(), req)
				if err != nil {
					return err
				}
				s.submitted = true
				path = res.Path
				return nil
			})
			if err != nil {
				log.Println(err)
				break // ずっとエラるようだったら諦める
			}
			if path == "" {
				break
			}

			s.chanTests <- &Test{
				Path:  path,
				Env:   []string{},
				Exec:  s.opts.Exec,
				Quiet: s.opts.Quiet,
				Merge: s.opts.Merge,
			}
		}
		close(s.chanTests)
		s.wgWorkers.Wait()
		close(s.chanSuites)
	}()

	for suite := range s.chanSuites {
		err = retry.Retry(s.opts.MaxRetry, s.opts.MaxDelay, func() error {
			_, err := client.Result(
				context.Background(),
				&ResultRequest{Path: suite.Path, Testsuite: suite.Suite},
			)
			if err != nil {
				log.Println(err)
			}
			return err
		})
		if err != nil {
			break // ずっとエラるようだったら諦める
		}
	}
}

// Find Test Files
func (s *Slave) findTestFiles() []string {
	files := []string{}
	if len(s.args) == 0 {
		files = s.appendFindTestFiles(files, "t")
	} else {
		for _, parent := range s.args {
			files = s.appendFindTestFiles(files, parent)
		}
	}
	return files
}

func (s *Slave) appendFindTestFiles(files []string, parent string) []string {
	stat, err := os.Stat(parent)
	if err != nil {
		panic(err)
	}
	if !stat.IsDir() {
		return append(files, parent)
	}

	filepath.Walk(parent, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".t") {
			files = append(files, path)
		}

		return nil
	})

	return files
}
