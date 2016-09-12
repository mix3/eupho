package eupho

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shogo82148/go-tap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Master struct {
	FlagSet *flag.FlagSet
	Addr    string
	Timeout time.Duration

	Formatter Formatter

	timeouter  *time.Timer
	testFiles  []string
	testResult map[string]*tap.Testsuite
	version    bool

	server     *grpc.Server
	testFileCh chan string
	endCh      chan error
	exitCode   int
	mu         sync.Mutex
}

func NewMaster() *Master {
	m := &Master{
		FlagSet:    flag.NewFlagSet("eupho", flag.ExitOnError),
		testResult: map[string]*tap.Testsuite{},
		testFileCh: make(chan string),
		endCh:      make(chan error),
		exitCode:   0,
	}
	m.FlagSet.StringVar(&m.Addr, "addr", "127.0.0.1:19300", "")
	m.FlagSet.DurationVar(&m.Timeout, "timeout", 10*time.Minute, "")
	m.FlagSet.BoolVar(&m.version, "version", false, "Show version of go-prove")
	return m
}

func (m *Master) ParseArgs(args []string) {
	m.FlagSet.Parse(args)

	m.timeouter = time.NewTimer(m.Timeout)
}

func (m *Master) Run(args []string) int {
	if args != nil {
		m.ParseArgs(args)
	}

	if m.version {
		fmt.Printf("eupho %s, %s built for %s/%s\n", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return m.exitCode
	}

	m.testFiles = m.findTestFiles()
	for _, file := range m.testFiles {
		m.testResult[file] = nil
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
	go func() {
		for _, path := range m.testFiles {
			m.testFileCh <- path
		}
		close(m.testFileCh)
	}()

	l, err := net.Listen("tcp", m.Addr)
	if err != nil {
		panic(fmt.Sprintf("failed to listen: %v", err))
	}
	m.server = grpc.NewServer()
	RegisterEuphoServer(m.server, m)
	log.Println("listen on", m.Addr)
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
	m.timeouter.Reset(m.Timeout)

	path := <-m.testFileCh
	if path != "" {
		log.Println("send:", path)
	}

	return &GetTestResponse{Path: path}, nil
}

func (m *Master) Result(ctx context.Context, req *ResultRequest) (*ResultResponse, error) {
	var ts tap.Testsuite
	if err := json.Unmarshal([]byte(req.Json), &ts); err != nil {
		return nil, err
	}
	log.Println("receive:", req.Path)
	m.EndCheck(req.Path, &ts)
	return &ResultResponse{}, nil
}

func (m *Master) EndCheck(path string, ts *tap.Testsuite) {
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

// Find Test Files
func (m *Master) findTestFiles() []string {
	files := []string{}
	if m.FlagSet.NArg() == 0 {
		files = m.appendFindTestFiles(files, "t")
	} else {
		for _, parent := range m.FlagSet.Args() {
			files = m.appendFindTestFiles(files, parent)
		}
	}
	return files
}

func (m *Master) appendFindTestFiles(files []string, parent string) []string {
	stat, err := os.Stat(parent)
	if err != nil {
		panic(err)
	}
	if !stat.IsDir() {
		return append(files, parent)
	}

	filepath.Walk(
		parent,
		func(path string, info os.FileInfo, err error) error {
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
