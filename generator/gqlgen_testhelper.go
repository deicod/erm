package generator

// SetGQLRunnerForTest overrides the gqlgen runner for tests and returns a restore function.
func SetGQLRunnerForTest(fn func(string) error) func() {
	prev := gqlRunner
	gqlRunner = fn
	return func() { gqlRunner = prev }
}
