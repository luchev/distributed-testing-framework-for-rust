package structs

// Error represents a test error by a name(category) and details
type Error struct {
	Name    string
	Details string
}

// TestResult represents a test with a name and an outcome
type TestResult struct {
	Name    string
	Passing bool
	Err     string
}

// Task represents a whole project with all tests
type Task struct {
	Name         string
	PassingBuild bool
	BuildMessage string
	Errors       []Error
	Tests        []TestResult
}

// Response represents the data provided to a go html template to render test results
type Response struct {
	PageTitle string
	Tasks     []Task
	Errors    []Error
}

// WorkerStatus represents the data provided to a go html template to render worker statuses
type WorkerStatus struct {
	URL    string
	Err    string
	Active bool
}
