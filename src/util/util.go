package util

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joshdk/go-junit"
	lg "github.com/luchev/dtf/logging"
	st "github.com/luchev/dtf/structs"
)

func RetrieveFile(request *http.Request, fieldName string) (string, string, error) {
	log.Printf("Starting file upload")

	file, handle, err := request.FormFile(fieldName)
	if err != nil {
		return "", "", err
	}

	fileName := handle.Filename

	defer file.Close()
	tempDir, err := ioutil.TempDir("uploads", "")
	if err != nil {
		return "", "", err
	}

	log.Printf("Uploading %s[%dm%s%s[%dm to %s[%dm%s%s[%dm\n",
		lg.Escape, lg.Underline, handle.Filename, lg.Escape, lg.Reset, lg.Escape, lg.Underline, tempDir, lg.Escape, lg.Reset)

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

func EscapeNewLineHTML(input string) string {
	return strings.Replace(input, "\n", "<br>", -1)
}

func Unzip(src string, dest string) ([]string, error) {
	log.Printf("Unzipping %s[%dm%s%s[%dm inside %s[%dm%s%s[%dm\n",
		lg.Escape, lg.Underline, src, lg.Escape, lg.Reset, lg.Escape, lg.Underline, dest, lg.Escape, lg.Reset)

	reader, err := zip.OpenReader(src)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer reader.Close()

	sourceFiles := make([]string, 0)
	for _, file := range reader.File {
		fpath := filepath.Join(dest, file.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return sourceFiles, fmt.Errorf("%s: illegal file path", fpath)
		}
		if file.FileInfo().IsDir() {
			log.Printf(" > Creating directory %s[%dm%s%s[%dm\n", lg.Escape, lg.Underline, dest+"/"+file.Name, lg.Escape, lg.Reset)
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return sourceFiles, err
		}

		log.Printf(" > Creating file %s[%dm%s%s[%dm\n", lg.Escape, lg.Underline, dest+"/"+file.Name, lg.Escape, lg.Reset)
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return sourceFiles, err
		}

		inFile, err := file.Open()
		if err != nil {
			return sourceFiles, err
		}

		if filepath.Ext(file.Name) == ".rs" {
			sourceFiles = append(sourceFiles, file.Name)
		}

		_, err = io.Copy(outFile, inFile)

		outFile.Close()
		inFile.Close()

		if err != nil {
			return sourceFiles, err
		}
	}

	return sourceFiles, nil
}

func GetTestNamesFromFiles(dir string, files []string) []string {
	log.Printf("Looking for tests in %s[%dm%s%s[%dm", lg.Escape, lg.Underline, dir, lg.Escape, lg.Reset)

	testNames := make([]string, 0)
	re := regexp.MustCompile(`\n*\s*#\[test\][\w\W]*?fn (\w+)`)
	for _, file := range files {
		content, err := ioutil.ReadFile(path.Join(dir, file))
		if err != nil {
			log.Printf("Failed to read file: %s[%dm%s%s[%dm: %s",
				lg.Escape, lg.Underline, file, lg.Escape, lg.Reset, err.Error())
			continue
		}

		matches := re.FindAllSubmatch(content, -1)
		for _, match := range matches {
			testNames = append(testNames, string(match[1]))
		}
	}

	return testNames
}

func RunTest(srcDir string, testName string) (st.TestResult, error) {
	log.Printf("Running test %s[%dm%s%s[%dm > %s[%dm%s%s[%dm\n",
		lg.Escape, lg.Underline, srcDir, lg.Escape, lg.Reset, lg.Escape, lg.Bold, testName, lg.Escape, lg.Reset)

	runTests := exec.Command("bash", "-c", fmt.Sprintf("cd %s ; cargo junit --test-name %s", srcDir, testName))
	var stdout bytes.Buffer
	runTests.Stdout = &stdout
	err := runTests.Run()

	suites, err := junit.Ingest(stdout.Bytes())
	if err != nil {
		return st.TestResult{}, fmt.Errorf(stdout.String())
	}

	// Should have exactly 1 test
	for _, suite := range suites {
		for _, test := range suite.Tests {
			result := st.TestResult{Name: test.Name, Passing: true, Err: ""}
			if test.Error != nil {
				result.Passing = false
				result.Err = test.Error.Error()
				log.Printf(" > %s[%dmFailure%s[%dm\n", lg.Escape, lg.FgRed, lg.Escape, lg.Reset)
			} else {
				log.Printf(" > %s[%dmSuccess%s[%dm\n", lg.Escape, lg.FgGreen, lg.Escape, lg.Reset)
			}

			return result, nil
		}
	}

	return st.TestResult{}, errors.New("Unexpected junit output")

}

func Build(srcDir string) (string, error) {
	log.Printf("Building %s[%dm%s%s[%dm", lg.Escape, lg.Bold, srcDir, lg.Escape, lg.Reset)

	cmd := exec.Command("bash", "-c", "cd "+srcDir+"; cargo build")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf(" > Build %s %s[%dmfailed%s[%dm\n", srcDir, lg.Escape, lg.FgRed, lg.Escape, lg.Reset)
	} else {
		log.Printf(" > Build %s %s[%dmsuccessful%s[%dm\n", srcDir, lg.Escape, lg.FgGreen, lg.Escape, lg.Reset)
	}

	return stderr.String(), err
}

func InitWorkspace(uploadDir string) {
	log.Println("Initializing workspace")

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		log.Printf(" > Creating %s[%dmuploads%s[%dm\n", lg.Escape, lg.Underline, lg.Escape, lg.Reset)
		err := os.Mkdir(uploadDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	dir, err := os.Open(uploadDir)
	if err != nil {
		log.Fatal(err)
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(" > Cleaning %s[%dmuploads%s[%dm\n", lg.Escape, lg.Underline, lg.Escape, lg.Reset)
	for _, fileName := range files {
		err = os.RemoveAll(filepath.Join(uploadDir, fileName))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func PingWorkers(workers map[string]struct{}) []st.WorkerStatus {
	statuses := make([]st.WorkerStatus, 0)
	log.Printf("Pinging workers")
	for worker := range workers {
		statuses = append(statuses, st.WorkerStatus{URL: worker, Err: "", Active: true})
		_, err := http.Get(worker + "/ping")
		if err != nil {
			log.Printf(" > Worker %s: %s[%dmdead%s[%dm\n", worker, lg.Escape, lg.FgRed, lg.Escape, lg.Reset)
			statuses[len(statuses)-1].Err = err.Error()
			statuses[len(statuses)-1].Active = false
		} else {
			log.Printf(" > Worker %s: %s[%dmalive%s[%dm\n", worker, lg.Escape, lg.FgGreen, lg.Escape, lg.Reset)
		}
	}

	return statuses
}

func SplitChunks(items []string, parts int) [][]string {
	min := func(a int, b int) int {
		if a < b {
			return a
		}
		return b
	}

	chunks := make([][]string, 0)
	limit := int(math.Ceil(float64(len(items)) / float64(parts)))
	for i := 0; i < len(items); i += limit {
		batch := items[i:min(i+limit, len(items))]
		chunks = append(chunks, batch)
	}

	return chunks
}

func GetActiveWorkers(workers map[string]struct{}) []string {
	activeWorkers := make([]string, 0)
	for _, worker := range PingWorkers(workers) {
		if worker.Active {
			activeWorkers = append(activeWorkers, worker.URL)
		}
	}
	return activeWorkers
}

func RunTests(names []string, srcDir string) (results []st.TestResult) {
	for _, test := range names {
		res, err := RunTest(srcDir, test)
		if err != nil {
			res = st.TestResult{Name: test, Passing: false, Err: fmt.Sprintf("Failed to run %s with err: %s", test, err.Error())}
			log.Printf("Failed to run %s: %s", test, err.Error())
		}
		results = append(results, res)
	}
	return results
}

func RunTestsRemotely(names []string, codeArchivePath string, worker string) (results []st.TestResult, err error) {
	worker += "/test"
	srcArchive, err := os.Open(codeArchivePath)
	if err != nil {
		return nil, err
	}
	defer srcArchive.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	testListWriter, err := writer.CreateFormField("testList")
	if err != nil {
		return nil, err
	}
	io.WriteString(testListWriter, strings.Join(names, ","))

	formFile, err := writer.CreateFormFile("codeZip", filepath.Base(srcArchive.Name()))
	if err != nil {
		return nil, err
	}

	io.Copy(formFile, srcArchive)
	writer.Close()
	request, err := http.NewRequest("POST", worker, body)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	responseString := string(responseBody)

	if response.StatusCode == http.StatusBadRequest {
		return nil, errors.New(responseString)
	}

	err = json.Unmarshal(responseBody, &results)
	if err != nil {
		return nil, err
	}

	return results, nil
}
