package eupho

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/mattn/go-shellwords"
	"github.com/mix3/pet"
)

type Test struct {
	Path  string
	Env   []string
	Exec  string
	Suite *pet.Testsuite
	Quiet bool
}

func (t *Test) Run() *pet.Testsuite {
	var (
		r io.Reader
		w io.WriteCloser
	)
	r, w = io.Pipe()
	if !t.Quiet {
		r = io.TeeReader(r, os.Stderr)
	}

	execParam, _ := shellwords.Parse(t.Exec)
	execParam = append(execParam, t.Path)

	cmd := exec.Command(execParam[0], execParam[1:]...)
	cmd.Env = t.Env
	cmd.Stdout = w
	cmd.Stderr = w

	suiteCh := make(chan *pet.Testsuite)
	go func() {
		var (
			suite       *pet.Testsuite
			parser, err = pet.NewParser(r)
		)
		if err != nil {
			suite = errorTestsuite(err)
		} else {
			suite, err = parser.Suite()
			if err != nil {
				suite = errorTestsuite(err)
			} else if suite.Plan < 0 {
				suite = errorTestsuite(
					fmt.Errorf("empty test. plan:%d", suite.Plan),
				)
			}
		}
		suiteCh <- suite
	}()

	cmd.Start()
	cmd.Wait()
	w.Close()

	t.Suite = <-suiteCh

	return t.Suite
}

func errorTestsuite(err error) *pet.Testsuite {
	return &pet.Testsuite{
		Ok: false,
		Tests: []*pet.Testline{
			&pet.Testline{
				Ok:          false,
				Num:         1,
				Description: "unexpected error",
				Diagnostic:  err.Error(),
			},
		},
		Plan:    1,
		Version: pet.DefaultTAPVersion,
	}
}
