package eupho

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/jessevdk/go-flags"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	pet "gopkg.in/mix3/pet.v1"
)

type Master struct {
	Formatter Formatter

	timeouter  *time.Timer
	testFiles  []string
	testResult map[string]*pet.Testsuite

	server     *grpc.Server
	testFileCh chan string
	endCh      chan error
	exitCode   int
	mu         sync.Mutex

	opts masterOptions
	args []string
}

type masterOptions struct {
	Addr    string        `          long:"addr"    default:"127.0.0.1:19300" description:"Listen addr"`
	Timeout time.Duration `          long:"timeout" default:"10m"             description:"Timeout duration"`
	Version bool          `          long:"version"                           description:"Show version of eupho"`
	Quiet   bool          `short:"q" long:"quiet"                             description:"quiet"`
}

func NewMaster() *Master {
	m := &Master{
		testResult: map[string]*pet.Testsuite{},
		testFileCh: make(chan string),
		endCh:      make(chan error),
		exitCode:   0,
	}
	return m
}

func (m *Master) ParseArgs(args []string) {
	var (
		opts masterOptions
		err  error
	)
	parser := flags.NewParser(
		&opts,
		flags.HelpFlag|flags.PassDoubleDash,
	)
	m.args, err = parser.ParseArgs(args)
	if err != nil {
		fmt.Println(err)
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	m.opts = opts

	m.timeouter = time.NewTimer(m.opts.Timeout)
}

func (m *Master) Run(args []string) int {
	if args != nil {
		m.ParseArgs(args)
	}

	if m.opts.Version {
		fmt.Printf("eupho %s, %s built for %s/%s\n", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return m.exitCode
	}

	m.startServe()

	if err := <-m.endCh; err != nil {
		panic(err)
	}

	m.stopServe()
	m.report()

	return m.exitCode
}

func (m *Master) startServe() {
	l, err := net.Listen("tcp", m.opts.Addr)
	if err != nil {
		panic(fmt.Sprintf("failed to listen: %v", err))
	}
	m.server = grpc.NewServer()
	RegisterEuphoServer(m.server, m)
	log.Println("listen on", m.opts.Addr)
	go m.server.Serve(l)

	go func() {
		<-m.timeouter.C
		m.endCh <- fmt.Errorf("slave request was lost")
	}()
}

func (m *Master) stopServe() {
	time.Sleep(1 * time.Second)
	m.server.Stop()
}

func (m *Master) report() {
	for path, suite := range m.testResult {
		m.Formatter.OpenTest(&Test{
			Path:  path,
			Suite: suite,
		})
	}
	m.Formatter.Report()
}

func (m *Master) GetTest(ctx context.Context, req *GetTestRequest) (*GetTestResponse, error) {
	m.initTestFiles(req.Submitted, req.TestFiles)

	m.timeouter.Reset(m.opts.Timeout)

	path := <-m.testFileCh
	if path != "" {
		peer, ok := peer.FromContext(ctx)
		if ok {
			log.Printf("send: %s -> %v", path, peer.Addr)
		} else {
			log.Printf("send: %s -> ???", path)
		}
	}

	return &GetTestResponse{Path: path}, nil
}

func (m *Master) Result(ctx context.Context, req *ResultRequest) (*ResultResponse, error) {
	var ts pet.Testsuite
	if err := json.Unmarshal([]byte(req.Json), &ts); err != nil {
		return nil, err
	}
	peer, ok := peer.FromContext(ctx)
	if ok {
		log.Printf("receive: %s <- %v", req.Path, peer.Addr)
	} else {
		log.Printf("receive: %s <- ???", req.Path)
	}
	if !m.opts.Quiet {
		for _, line := range ts.Tests {
			if !line.Ok {
				fmt.Fprintln(os.Stderr, line.WalkDiagnostic())
			}
		}
	}
	m.EndCheck(req.Path, &ts)
	return &ResultResponse{}, nil
}

func (m *Master) EndCheck(path string, ts *pet.Testsuite) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.testResult[path] = ts
	if !ts.Ok {
		m.exitCode = 1
	}
	for _, tr := range m.testResult {
		if tr == nil {
			return
		}
	}
	m.endCh <- nil
}

func (m *Master) initTestFiles(submitted bool, testFiles []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if submitted {
		return
	}

	if m.testFiles != nil {
		return
	}

	go func() {
		for _, path := range m.testFiles {
			m.testFileCh <- path
		}
		close(m.testFileCh)
	}()

	m.testFiles = []string{}
	for _, f := range testFiles {
		if _, ok := m.testResult[f]; ok {
			continue
		}

		m.testFiles = append(m.testFiles, f)
		m.testResult[f] = nil
	}

	if len(m.testFiles) == 0 {
		m.endCh <- nil
	}
}
