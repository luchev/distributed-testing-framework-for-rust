package task

import (
	"bytes"
	"log"
	"os/exec"

	"github.com/joshdk/go-junit"
	"github.com/luchev/dtf/structs/error"
	"github.com/luchev/dtf/structs/test"
)

type Task struct {
	Name         string  `yaml:"name"`
	TestScript   string  `yaml:"testScript"`
	MemoryScript string  `yaml:"memoryScript"`
	MemoryPoints float64 `yaml:"memoryPoints"`
}

func (t *Task) RunTestScript() (result TaskResult) {
	cmd := exec.Command("bash", "-c", t.TestScript)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		result.PassingBuild = false
		result.Errors = append(result.Errors, error.Error{Name: "Build error", Details: err.Error()})
	} else {
		suites, err := junit.Ingest(stdout.Bytes())
		if err != nil {
			log.Fatal("Failed to parse JUnit output")
		}
		result.PassingBuild = true
		result.Tests, result.Points = test.ParseJunitTests(suites)
	}
	return
}

func (t *Task) HasMemoryLeak() bool {
	cmd := exec.Command("bash", "-c", t.MemoryScript)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err == nil {
		return false
	} else {
		return true
	}
}
