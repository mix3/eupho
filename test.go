package eupho

import (
	"io"
	"os"
	"os/exec"

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
	cmd.Wait()
	w.Close()
	r.Close()

	suite := <-ch
	t.Suite = suite
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
