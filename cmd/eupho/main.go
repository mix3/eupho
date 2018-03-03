package main

import (
	"os"

	"github.com/mix3/eupho"
)

func main() {
	m := eupho.NewMaster()
	m.ParseArgs(os.Args[1:])
	os.Exit(m.Run(nil))
}
