package util

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	lg "github.com/luchev/dtf/logging"
	"gopkg.in/yaml.v2"
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

func RetrieveLocalFile(path string) (string, string, error) {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	tempDir, err := ioutil.TempDir("uploads", "")
	if err != nil {
		return "", "", err
	}

	fileName := filepath.Base(path)
	err = ioutil.WriteFile(tempDir+"/"+fileName, input, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	return tempDir, fileName, nil
}

func EscapeNewLineHTML(input string) string {
	return strings.Replace(input, "\n", "<br>", -1)
}

func Unzip(src string, dest string) ([]string, error) {
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
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return sourceFiles, err
		}

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

func ExecuteScript(path string) (string, error) {
	base := filepath.Base(path)
	dir := filepath.Dir(path)
	cmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s; ./%s", dir, base))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stderr.String(), err
	} else {
		return stdout.String(), nil
	}
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
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
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

	for _, fileName := range files {
		err = os.RemoveAll(filepath.Join(uploadDir, fileName))
		if err != nil {
			log.Fatal(err)
		}
	}
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

func UnmarshalYamlFile(path string, out interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	bytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(bytes, out)
	if err != nil {
		return err
	}

	return nil
}
