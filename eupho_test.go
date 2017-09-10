package eupho_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/mix3/eupho"
	"github.com/mix3/eupho/formatter"
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
			dir,
		})
		s.Run(nil)
	}()

	m := eupho.NewMaster()
	m.Formatter = &formatter.JUnitFormatter{}
	m.ParseArgs([]string{
		"--addr", addr,
	})
	code := 0
	out := captureStdout(func() {
		code = m.Run(nil)
	})
	if code != 0 {
		t.Errorf("ExitCode want 0, but got %d\n", code)
	}

	//<?xml version="1.0" encoding="UTF-8"?>
	//<testsuites>
	//    <testsuite tests="1" failures="0" errors="0" skipped="0" time="0.302" name="_var_folders_qf_mkfd3g0n6zn71mmr84g_qhg80000gp_T_130265267_01_t">
	//        <properties></properties>
	//        <testcase classname="_var_folders_qf_mkfd3g0n6zn71mmr84g_qhg80000gp_T_130265267_01_t" name="" time="0.302">
	//            <system-out><![CDATA[ok 1
	//]]></system-out>
	//        </testcase>
	//    </testsuite>
	//</testsuites>

	ok, err := regexp.Match(`<testsuites>
    <testsuite tests="1" failures="0" errors="0" skipped="0" time="[0-9\.]+" name="[^"]*">
        <properties></properties>
        <testcase classname="[^"]*" name="" time="[0-9\.]+">
            <system-out><!\[CDATA\[ok 1
\]\]></system-out>
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
			dir,
		})
		s.Run(nil)
	}()

	m := eupho.NewMaster()
	m.Formatter = &formatter.JUnitFormatter{}
	m.ParseArgs([]string{
		"--addr", addr,
	})
	code := 1
	out := captureStdout(func() {
		code = m.Run(nil)
	})
	if code != 1 {
		t.Errorf("ExitCode want 1, but got %d\n", code)
	}

	//<testsuites>
	//    <testsuite tests="1" failures="1" errors="0" skipped="0" time="0.094" name="_var_folders_qf_mkfd3g0n6zn71mmr84g_qhg80000gp_T_778020036_01_t">
	//        <properties></properties>
	//        <testcase classname="_var_folders_qf_mkfd3g0n6zn71mmr84g_qhg80000gp_T_778020036_01_t" name="" time="0.094">
	//            <failure message="not ok 1" type="TestFailed"></failure>
	//            <system-out><![CDATA[not ok 1
	//]]></system-out>
	//        </testcase>
	//    </testsuite>
	//</testsuites>

	ok, err := regexp.Match(`<testsuites>
    <testsuite tests="1" failures="1" errors="0" skipped="0" time="[0-9\.]+" name="[^"]*">
        <properties></properties>
        <testcase classname="[^"]*" name="" time="[0-9\.]+">
            <failure message="not ok 1" type="TestFailed"></failure>
            <system-out><!\[CDATA\[not ok 1
\]\]></system-out>
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
	out := captureStdout(func() {
		code = s.Run(nil)
	})
	if code != 0 {
		t.Errorf("ExitCode want 0, but got %d\n", code)
	}

	//<testsuites>
	//    <testsuite tests="1" failures="0" errors="0" skipped="0" time="0.106" name="_var_folders_qf_mkfd3g0n6zn71mmr84g_qhg80000gp_T_100549956_01_t">
	//        <properties></properties>
	//        <testcase classname="_var_folders_qf_mkfd3g0n6zn71mmr84g_qhg80000gp_T_100549956_01_t" name="" time="0.106">
	//            <system-out><![CDATA[ok 1
	//]]></system-out>
	//        </testcase>
	//    </testsuite>
	//</testsuites>

	ok, err := regexp.Match(`<testsuites>
    <testsuite tests="1" failures="0" errors="0" skipped="0" time="[0-9\.]+" name="[^"]*">
        <properties></properties>
        <testcase classname="[^"]*" name="" time="[0-9\.]+">
            <system-out><!\[CDATA\[ok 1
\]\]></system-out>
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
	out := captureStdout(func() {
		code = s.Run(nil)
	})
	if code != 1 {
		t.Errorf("ExitCode want 1, but got %d\n", code)
	}

	//<testsuites>
	//    <testsuite tests="1" failures="1" errors="0" skipped="0" time="0.094" name="_var_folders_qf_mkfd3g0n6zn71mmr84g_qhg80000gp_T_340899990_01_t">
	//        <properties></properties>
	//        <testcase classname="_var_folders_qf_mkfd3g0n6zn71mmr84g_qhg80000gp_T_340899990_01_t" name="" time="0.094">
	//            <failure message="not ok 1" type="TestFailed"></failure>
	//            <system-out><![CDATA[not ok 1
	//]]></system-out>
	//        </testcase>
	//    </testsuite>
	//</testsuites>

	ok, err := regexp.Match(`<testsuites>
    <testsuite tests="1" failures="1" errors="0" skipped="0" time="[0-9\.]+" name="[^"]*">
        <properties></properties>
        <testcase classname="[^"]*" name="" time="[0-9\.]+">
            <failure message="not ok 1" type="TestFailed"></failure>
            <system-out><!\[CDATA\[not ok 1
\]\]></system-out>
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

// https://gist.github.com/mindscratch/0faa78bd3c0005d080bf
// not thread safe
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
