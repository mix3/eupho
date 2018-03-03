package formatter

import (
	"fmt"

	"github.com/mix3/eupho/test"
	pet "gopkg.in/mix3/pet.v3"
)

type TapFormatter struct {
	Suites []*pet.Testsuite
}

func (f *TapFormatter) OpenTest(test *test.Test) {
	f.Suites = append(f.Suites, test.Suite)
}

func (f *TapFormatter) Report() {
	for _, s := range f.Suites {
		for _, t := range s.Tests {
			fmt.Printf("%#v", t)
		}
	}
}
