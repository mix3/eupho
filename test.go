package eupho

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/mattn/go-shellwords"
	pet "gopkg.in/mix3/pet.v3"
)

type Test struct {
	Path string
	Env  []string
	Exec string

	// Merge test scripts' STDERR with their STDOUT.
	Merge bool

	Suite *pet.Testsuite
	Quiet bool
}

func (t *Test) Run() *pet.Testsuite {
	execParam, _ := shellwords.Parse(t.Exec)
	execParam = append(execParam, t.Path)
	cmd := exec.Command(execParam[0], execParam[1:]...)
	cmd.Env = t.Env

	r, w := io.Pipe()
	cmd.Stdout = w

	if t.Merge {
		cmd.Stderr = w
	} else {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		t.Suite = errorTestsuite(err)
		return t.Suite
	}

	ch := make(chan *pet.Testsuite)
	go func() {
		parser, err := pet.NewParser(r)
		if err != nil {
			ch <- errorTestsuite(err)
			return
		}
		suite, err := parser.Suite()
		if err != nil {
			ch <- errorTestsuite(err)
			return
		}
		ch <- suite
	}()

	err := cmd.Wait()
	w.Close()
	r.Close()

	suite := <-ch
	t.Suite = suite

	if err == nil {
		return suite
	}

	// check exit code
	if err == nil {
		return suite
	}
	// from http://qiita.com/hnakamur/items/5e6f22bda8334e190f63
	if e2, ok := err.(*exec.ExitError); ok {
		if s, ok := e2.Sys().(syscall.WaitStatus); ok {
			exitCode := s.ExitStatus()
			if exitCode != 0 {
				suite.Ok = false
				suite.Plan++
				suite.Tests = append(suite.Tests, &pet.Testline{
					Ok:          false,
					Num:         suite.Plan,
					Description: fmt.Sprintf("Test died with return code %d", exitCode),
				})
			}
			return suite
		}
	}

	suite.Ok = false
	suite.Plan++
	suite.Tests = append(suite.Tests, &pet.Testline{
		Ok:          false,
		Num:         suite.Plan,
		Description: "unexpected error",
		Diagnostic:  err.Error(),
	})

	return suite
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
