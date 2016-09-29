package eupho

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestRun_success(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`print "1..1\nok 1\n";`)

	test := &Test{
		Path: f.Name(),
		Env:  os.Environ(),
		Exec: "perl",
	}

	suite := test.Run()
	if suite == nil {
		t.Error("want not nil\ngot nil")
	}
	if test.Suite != suite {
		t.Errorf("want that %v equals to %v", test.Suite, suite)
	}

	if !suite.Ok {
		t.Error("want success\ngot fail")
	}
	if len(suite.Tests) != 1 {
		t.Errorf("want 1\ngot %d", len(suite.Tests))
	}
}

func TestRun_fail(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`print "1..1\nnot ok 1\n";`)

	test := &Test{
		Path: f.Name(),
		Env:  os.Environ(),
		Exec: "perl",
	}

	suite := test.Run()
	if suite == nil {
		t.Error("want not nil\ngot nil")
	}
	if test.Suite != suite {
		t.Errorf("want that %v equals to %v", test.Suite, suite)
	}

	if suite.Ok {
		t.Error("want fail\ngot success")
	}
	if len(suite.Tests) != 1 {
		t.Errorf("want 1\ngot %d", len(suite.Tests))
	}
}

func TestRun_failplan(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`print "1..2\nok 1\n";`)

	test := &Test{
		Path: f.Name(),
		Env:  os.Environ(),
		Exec: "perl",
	}

	suite := test.Run()
	if suite == nil {
		t.Error("want not nil\ngot nil")
	}
	if test.Suite != suite {
		t.Errorf("want that %v equals to %v", test.Suite, suite)
	}

	if suite.Ok {
		t.Error("want fail\ngot success")
	}
	if len(suite.Tests) != 1 {
		t.Errorf("want 1\ngot %d", len(suite.Tests))
	}
}

func TestRun_empty(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`die "test failed!!!";`)

	test := &Test{
		Path: f.Name(),
		Env:  os.Environ(),
		Exec: "perl",
	}

	suite := test.Run()
	if suite == nil {
		t.Error("want not nil\ngot nil")
	}
	if test.Suite != suite {
		t.Errorf("want that %v equals to %v", test.Suite, suite)
	}

	if suite.Ok {
		t.Error("want fail\ngot success")
	}
	if len(suite.Tests) != 1 {
		t.Errorf("want 1\ngot %d", len(suite.Tests))
	}
}

func TestRun_subtest(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`
use Test::More;
subtest hoge => sub {
	subtest fuga => sub {
		subtest piyo => sub {
			ok 1;
			ok 1;
			ok 1;
		};
	};
};
done_testing;
`)

	test := &Test{
		Path: f.Name(),
		Env:  os.Environ(),
		Exec: "perl",
	}

	suite := test.Run()
	if suite == nil {
		t.Error("want not nil\ngot nil")
	}
	if test.Suite != suite {
		t.Errorf("want that %v equals to %v", test.Suite, suite)
	}

	if !suite.Ok {
		t.Error("want success\ngot fail")
	}
	if len(suite.Tests) != 1 {
		t.Errorf("want 1\ngot %d", len(suite.Tests))
	}
}
