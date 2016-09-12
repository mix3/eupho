package plugin

import (
	"fmt"
	"log"

	"github.com/lestrrat/go-test-mysqld"
	"github.com/mix3/eupho"
)

type TestMysqld struct{}

func init() {
	eupho.AppendPluginLoader(
		"mysqld",
		eupho.PluginLoaderFunc(
			func(name, args string) eupho.Plugin {
				return &TestMysqld{}
			},
		),
	)
}

func (p *TestMysqld) Run(w *eupho.Worker, f func()) {
	log.Printf("run mysqld")
	mysqld, err := mysqltest.NewMysqld(nil)
	if err != nil {
		log.Printf("mysql error: %s\n", err)
	}
	defer mysqld.Stop()

	address := mysqld.ConnectString(0)
	w.Env = append(w.Env, fmt.Sprintf("GO_PROVE_MYSQLD=%s", address))

	f()
}
