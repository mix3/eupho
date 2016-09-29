package eupho

type Formatter interface {
	// Called to create a new test
	OpenTest(test *Test)

	// Prints the report after all tests are run
	Report()
}
