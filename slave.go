package eupho

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Songmu/retry"
	"github.com/soh335/sliceflag"
	"google.golang.org/grpc"
)

type Slave struct {
	FlagSet *flag.FlagSet

	Addr    string
	Jobs    int
	Exec    string
	Plugins []Plugin

	chanTests  chan *Test
	chanSuites chan *Test
	wgWorkers  *sync.WaitGroup
	pluginArgs []string
	version    bool
	maxDelay   time.Duration
	maxRetry   uint
}

func NewSlave() *Slave {
	s := &Slave{
		FlagSet:    flag.NewFlagSet("eupho", flag.ExitOnError),
		Plugins:    []Plugin{},
		chanTests:  make(chan *Test),
		chanSuites: make(chan *Test),
		wgWorkers:  &sync.WaitGroup{},
	}
	s.FlagSet.IntVar(&s.Jobs, "j", 1, "")
	s.FlagSet.IntVar(&s.Jobs, "jobs", 1, "")
	s.FlagSet.StringVar(&s.Addr, "addr", "127.0.0.1:19300", "")
	s.FlagSet.StringVar(&s.Exec, "exec", "perl", "")
	s.FlagSet.BoolVar(&s.version, "version", false, "Show version of go-prove")
	s.FlagSet.DurationVar(&s.maxDelay, "max-delay", 3*time.Second, "")
	s.FlagSet.UintVar(&s.maxRetry, "max-retry", 10, "")
	sliceflag.StringVar(s.FlagSet, &s.pluginArgs, "plugin", []string{}, "plugins")
	sliceflag.StringVar(s.FlagSet, &s.pluginArgs, "P", []string{}, "plugins")
	return s
}

func (s *Slave) ParseArgs(args []string) {
	s.FlagSet.Parse(args)

	if s.Jobs < 1 {
		s.Jobs = 1
	}

	for _, plugin := range s.pluginArgs {
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

	if s.version {
		fmt.Printf("eupho %s, %s built for %s/%s\n", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return
	}

	for i := 0; i < s.Jobs; i++ {
		w := NewWorker(s)
		w.Start()
	}

	conn, err := grpc.Dial(
		s.Addr,
		grpc.WithInsecure(),
		grpc.WithBackoffMaxDelay(s.maxDelay),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := NewEuphoClient(conn)

	go func() {
		for {
			var path string
			if err := retry.Retry(s.maxRetry, s.maxDelay, func() error {
				res, err := client.GetTest(context.Background(), &GetTestRequest{})
				if err != nil {
					return err
				}
				path = res.Path
				return nil
			}); err != nil {
				log.Println(err)
				break // ずっとエラるようだったら諦める
			}
			if path == "" {
				break
			}

			s.chanTests <- &Test{
				Path: path,
				Env:  []string{},
				Exec: s.Exec,
			}
		}
		close(s.chanTests)
		s.wgWorkers.Wait()
		close(s.chanSuites)
	}()

	for suite := range s.chanSuites {
		b, err := json.Marshal(suite.Suite)
		if err != nil {
			panic(err)
		}
		if err := retry.Retry(s.maxRetry, s.maxDelay, func() error {
			_, err := client.Result(
				context.Background(),
				&ResultRequest{
					Path: suite.Path,
					Json: string(b),
				},
			)
			if err != nil {
				log.Println(err)
			}
			return err
		}); err != nil {
			break // ずっとエラるようだったら諦める
		}
	}
}
