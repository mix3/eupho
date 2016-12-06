package eupho_test

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/mix3/eupho"
	"github.com/mix3/eupho/formatter"
	"github.com/sevagh/stdcap"
)

func newTempFiles(files map[string]string) (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	for name, content := range files {
		tmpFn := filepath.Join(dir, name)
		if err := ioutil.WriteFile(tmpFn, []byte(content), 0644); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}

func TestMasterSlaveOK(t *testing.T) {
	dir, err := newTempFiles(map[string]string{
		`01.t`: `use Test::More;
ok 1;
done_testing;`,
	})
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(dir)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Error(err)
		return
	}
	addr := l.Addr().String()
	l.Close()

	go func() {
		s := eupho.NewSlave()
		s.ParseArgs([]string{
			"--addr", addr,
		})
		s.Run(nil)
	}()

	m := eupho.NewMaster()
	m.Formatter = &formatter.JUnitFormatter{}
	m.ParseArgs([]string{
		"--addr", addr,
		dir,
	})
	code := 0
	sc := stdcap.StdoutCapture()
	out := sc.Capture(func() {
		code = m.Run(nil)
	})
	if code != 0 {
		t.Errorf("ExitCode want 0, but got %d\n", code)
	}

	ok, err := regexp.Match(`<testsuites>
	<testsuite tests="1" failures="0" time="[0-9\.]+" name="[^"]*">
		<properties></properties>
		<testcase classname="[^"]*" name="" time="[0-9\.]+"></testcase>
	</testsuite>
</testsuites>`, []byte(out))
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Errorf("incorrect output\n%s", out)
	}
}

func TestMasterSlaveFail(t *testing.T) {
	dir, err := newTempFiles(map[string]string{
		`01.t`: `use Test::More;
ok 0;
done_testing;`,
	})
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(dir)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Error(err)
		return
	}
	addr := l.Addr().String()
	l.Close()

	go func() {
		s := eupho.NewSlave()
		s.ParseArgs([]string{
			"--addr", addr,
		})
		s.Run(nil)
	}()

	m := eupho.NewMaster()
	m.Formatter = &formatter.JUnitFormatter{}
	m.ParseArgs([]string{
		"--addr", addr,
		dir,
	})
	code := 1
	sc := stdcap.StdoutCapture()
	out := sc.Capture(func() {
		code = m.Run(nil)
	})
	if code != 1 {
		t.Errorf("ExitCode want 1, but got %d\n", code)
	}
	ok, err := regexp.Match(`<testsuites>
	<testsuite tests="1" failures="1" time="[0-9\.]+" name="[^"]*">
		<properties></properties>
		<testcase classname="[^"]*" name="" time="[0-9\.]+">
			<failure message="not ok" type=""></failure>
		</testcase>
	</testsuite>
</testsuites>`, []byte(out))
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Errorf("incorrect output\n%s", out)
	}
}

func TestSoloOK(t *testing.T) {
	dir, err := newTempFiles(map[string]string{
		`01.t`: `use Test::More;
ok 1;
done_testing;`,
	})
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(dir)

	s := eupho.NewSolo()
	s.Master.Formatter = &formatter.JUnitFormatter{}
	s.ParseArgs([]string{
		dir,
	})
	code := 0
	sc := stdcap.StdoutCapture()
	out := sc.Capture(func() {
		code = s.Run(nil)
	})
	if code != 0 {
		t.Errorf("ExitCode want 0, but got %d\n", code)
	}

	ok, err := regexp.Match(`<testsuites>
	<testsuite tests="1" failures="0" time="[0-9\.]+" name="[^"]*">
		<properties></properties>
		<testcase classname="[^"]*" name="" time="[0-9\.]+"></testcase>
	</testsuite>
</testsuites>`, []byte(out))
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Errorf("incorrect output\n%s", out)
	}
}

func TestSoloFail(t *testing.T) {
	dir, err := newTempFiles(map[string]string{
		`01.t`: `use Test::More;
ok 0;
done_testing;`,
	})
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(dir)

	s := eupho.NewSolo()
	s.Master.Formatter = &formatter.JUnitFormatter{}
	s.ParseArgs([]string{
		dir,
	})
	code := 1
	sc := stdcap.StdoutCapture()
	out := sc.Capture(func() {
		code = s.Run(nil)
	})
	if code != 1 {
		t.Errorf("ExitCode want 1, but got %d\n", code)
	}

	ok, err := regexp.Match(`<testsuites>
	<testsuite tests="1" failures="1" time="[0-9\.]+" name="[^"]*">
		<properties></properties>
		<testcase classname="[^"]*" name="" time="[0-9\.]+">
			<failure message="not ok" type=""></failure>
		</testcase>
	</testsuite>
</testsuites>`, []byte(out))
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Errorf("incorrect output\n%s", out)
	}
}
