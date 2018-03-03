package eupho

import "github.com/mix3/eupho/test"

type Formatter interface {
	// Called to create a new test
	OpenTest(test *test.Test)

	// Prints the report after all tests are run
	Report()
}
