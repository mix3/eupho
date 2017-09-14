package eupho

import (
	"fmt"
	"log"
	"os"
	"sync"
)

type Worker struct {
	slave *Slave
	Env   []string
	wg    sync.WaitGroup
}

func NewWorker(slave *Slave, id int) *Worker {
	env := append(os.Environ(), fmt.Sprintf("GO_PROVE_WORKER_ID=%d", id))
	return &Worker{
		Env:   env,
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
	}

	for _, p := range w.slave.Plugins {
		pp := p
		g := f
		f = func() {
			pp.Run(w, g)
		}
	}

	f()
	w.slave.wgWorkers.Done()
}
