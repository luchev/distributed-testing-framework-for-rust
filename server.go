package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/joshdk/go-junit"
)

const uploadDir = "uploads"
const buildScriptName = "build.sh"

func retrieveFile(request *http.Request) (string, string, error) {
	fmt.Println("Uploading file...")
	request.ParseMultipartForm(10 << 20) // 10 MB files

	file, handle, err := request.FormFile("codeZip")

	if err != nil {
		return "", "", err
	}

	fileName := handle.Filename

	defer file.Close()
	tempDir, err := ioutil.TempDir("uploads", "")
	if err != nil {
		return "", "", err
	}

	archiveFile, err := os.OpenFile(tempDir+"/"+fileName, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	uploadedBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", "", nil
	}

	archiveFile.Write(uploadedBytes)

	return tempDir, fileName, nil
}

// Error TODO
type Error struct {
	Name    string
	Details string
}

// TestResult TODO
type TestResult struct {
	Name    string
	Passing bool
	Err     string
}

// Task TODO
type Task struct {
	Name         string
	PassingBuild bool
	Errors       []Error
	Tests        []TestResult
}

// Response TODO
type Response struct {
	PageTitle string
	Tasks     []Task
	Errors    []Error
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	response := Response{"File name error", nil, nil}
	problemSetID := mux.Vars(r)["problemset"]

	defer func() {
		tmpl, _ := template.New("homework.gohtml").Funcs(template.FuncMap{"escapeNewLineHTML": escapeNewLineHTML}).ParseFiles("templates/homework.gohtml")
		tmpl.Execute(w, response)
	}()

	tempDir, fileName, err := retrieveFile(r)
	if err != nil {
		response.Errors = append(response.Errors, Error{"Failed upload", err.Error()})
		return
	}
	response.PageTitle = fileName
	// defer os.RemoveAll(tempDir)

	taskDirs, err := unzip(tempDir+"/"+fileName, tempDir)
	if err != nil {
		response.Errors = append(response.Errors, Error{"Failed extract", err.Error()})
		return
	}

	sort.Strings(taskDirs)

	// Build
	for _, dir := range taskDirs {
		response.Tasks = append(response.Tasks, Task{filepath.Base(dir), true, nil, nil})
		output, err := build(dir)
		if err != nil {
			response.Tasks[len(response.Tasks)-1].PassingBuild = false
			response.Tasks[len(response.Tasks)-1].Errors =
				append(response.Tasks[len(response.Tasks)-1].Errors, Error{"Failed build", output})
		}
	}

	// Test
	for index, dir := range taskDirs {
		if response.Tasks[index].PassingBuild {
			testResults, err := test(problemSetID, filepath.Base(dir), dir)
			if err != nil {
				response.Tasks[index].Errors = append(response.Tasks[index].Errors, Error{"Problem 404", err.Error()})
			}
			response.Tasks[index].Tests = testResults
		}
	}
}

func test(problemSet string, problem string, codeDir string) ([]TestResult, error) {
	testDir := fmt.Sprintf("tests/%s/%s", problemSet, problem)

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("Incorrect problem name: %s. Or redundant directory in zip archive", problem)
	}

	copyTests := exec.Command("bash", "-c", fmt.Sprintf("cp -r tests/%s/%s/* %s/", problemSet, problem, codeDir))
	copyTests.Run()

	runTests := exec.Command("bash", "-c", fmt.Sprintf("cd %s ; bash %s", codeDir, buildScriptName))
	var stdout bytes.Buffer
	runTests.Stdout = &stdout
	err := runTests.Run()

	suites, err := junit.Ingest(stdout.Bytes())
	if err != nil {
		return nil, fmt.Errorf(stdout.String())
	}

	testResults := make([]TestResult, 0)
	for _, suite := range suites {
		for _, test := range suite.Tests {
			result := TestResult{test.Name, true, ""}
			if test.Error != nil {
				result.Passing = false
				result.Err = test.Error.Error()
			}
			testResults = append(testResults, result)
		}
	}

	return testResults, nil
}

func setupRoutes() {
	router := mux.NewRouter()
	router.HandleFunc("/problem/{problemset}", uploadFile)
	srv := http.Server{
		Handler:      router,
		Addr:         "127.0.0.1:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func initWorkspace() error {
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err := os.Mkdir(uploadDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	dir, err := os.Open(uploadDir)
	if err != nil {
		return err
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, fileName := range files {
		err = os.RemoveAll(filepath.Join(uploadDir, fileName))
		if err != nil {
			return nil
		}
	}
	return nil
}

func unzip(src string, dest string) ([]string, error) {
	var subdirs []string

	reader, err := zip.OpenReader(src)
	if err != nil {
		return subdirs, err
	}
	defer reader.Close()

	for _, file := range reader.File {
		fpath := filepath.Join(dest, file.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return subdirs, fmt.Errorf("%s: illegal file path", fpath)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			if filepath.Dir(fpath) == dest {
				subdirs = append(subdirs, fpath)
			}
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return subdirs, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return subdirs, err
		}

		inFile, err := file.Open()
		if err != nil {
			return subdirs, err
		}

		_, err = io.Copy(outFile, inFile)

		outFile.Close()
		inFile.Close()

		if err != nil {
			return subdirs, err
		}
	}

	return subdirs, nil
}

func escapeNewLineHTML(input string) string {
	return strings.Replace(input, "\n", "<br>", -1)
}

func build(dir string) (string, error) {
	cmd := exec.Command("bash", "-c", "shopt -s nullglob; shopt -s globstar; cd "+dir+"; clang++ **/*.cpp **/*.hpp -c -Wall -Wextra -Werror")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stderr.String(), err
}

func main() {
	fmt.Println("Starting server ...")
	err := initWorkspace()
	if err != nil {
		fmt.Println(err)
	}
	setupRoutes()
}
