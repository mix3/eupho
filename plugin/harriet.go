package plugin

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/mix3/eupho"
)

type Harriet struct {
	cmd  string
	args []string
}

func init() {
	eupho.AppendPluginLoader("harriet", eupho.PluginLoaderFunc(harrietLoader))
}

func harrietLoader(name, args string) eupho.Plugin {
	cmd := "harriet"
	cmdArgs := []string{"t/harriet"}

	a, _ := shellwords.Parse(args)
	if len(a) > 0 {
		cmd = a[0]
		cmdArgs = a[1:]
	}
	return &Harriet{
		cmd:  cmd,
		args: cmdArgs,
	}
}

func (p *Harriet) Run(w *eupho.Worker, f func()) {
	log.Printf("run harriet cmd: %s %s", p.cmd, p.args)

	cmd := exec.Command(p.cmd, p.args...)
	cmd.Env = w.Env

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	cmd.Start()
	go io.Copy(os.Stderr, stderr)

	s := bufio.NewScanner(stdout)
	foundExport := false
	for s.Scan() {
		t := s.Text()
		if strings.HasPrefix(t, "export") {
			exportCmd, _ := shellwords.Parse(t)
			if len(exportCmd) < 2 {
				continue
			}
			log.Printf("export %s", exportCmd[1])
			w.Env = append(w.Env, exportCmd[1])
			foundExport = true
		}
		if foundExport && t == "" {
			break
		}
	}

	defer func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	}()

	f()
}
