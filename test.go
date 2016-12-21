package eupho

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/mattn/go-shellwords"
	pet "gopkg.in/mix3/pet.v1"
)

type Test struct {
	Path  string
	Env   []string
	Exec  string
	Suite *pet.Testsuite
	Quiet bool
}

func (t *Test) Run() *pet.Testsuite {
	execParam, _ := shellwords.Parse(t.Exec)
	execParam = append(execParam, t.Path)
	cmd := exec.Command(execParam[0], execParam[1:]...)
	cmd.Env = t.Env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Suite = errorTestsuite(err)
		return t.Suite
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Suite = errorTestsuite(err)
		return t.Suite
	}

	err = cmd.Start()
	if err != nil {
		t.Suite = errorTestsuite(err)
		return t.Suite
	}
	go io.Copy(os.Stderr, stderr)

	suiteCh := make(chan *pet.Testsuite)
	go func() {
		var (
			suite       *pet.Testsuite
			parser, err = pet.NewParser(stdout)
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
	cmd.Wait()

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
