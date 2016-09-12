package eupho

import (
	"log"
	"os"
	"sync"
)

type Worker struct {
	slave *Slave
	Env   []string
	wg    sync.WaitGroup
}

func NewWorker(slave *Slave) *Worker {
	return &Worker{
		Env:   os.Environ(),
		slave: slave,
	}
}

func (w *Worker) Start() {
	w.slave.wgWorkers.Add(1)
	go w.run()
}

func (w *Worker) run() {
	f := func() {
		for test := range w.slave.chanTests {
			test.Env = w.Env
			log.Printf("start %s", test.Path)
			test.Run()
			w.slave.chanSuites <- test
			log.Printf("finish %s", test.Path)
		}
		w.slave.wgWorkers.Done()
	}

	for _, p := range w.slave.Plugins {
		f = func(g func()) func() {
			return func() { p.Run(w, g) }
		}(f)
	}

	f()
}
