package eupho

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/mix3/eupho/test"
)

type testPlugin int

var pluginResChan chan int

func (p testPlugin) Run(w *Worker, f func()) {
	pluginResChan <- int(p)
	f()
	pluginResChan <- int(p)
}

func Test__run(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`print "1..1\nok 1\n";`)

	tt := &test.Test{
		Path: f.Name(),
		Env:  os.Environ(),
		Exec: "perl",
	}

	pluginResChan = make(chan int, 4)

	s := NewSlave()
	w := NewWorker(s, 0)
	s.Plugins = []Plugin{
		testPlugin(1),
		testPlugin(2),
	}

	w.Start()

	sendCh := make(chan *test.Test)
	s.chanTests <- sendCh
	sendCh <- tt
	go func() {
		for range s.chanSuites {
		}
	}()
	close(s.chanTests)
	s.wgWorkers.Wait()
	close(pluginResChan)

	pluginRes := make([]int, 0, 4)
	for res := range pluginResChan {
		pluginRes = append(pluginRes, res)
	}
	if !reflect.DeepEqual(pluginRes, []int{2, 1, 1, 2}) {
		t.Errorf(
			"plugin exec is not valid: got: %v, expect: [2 1 1 2]",
			pluginRes,
		)
	}
}
