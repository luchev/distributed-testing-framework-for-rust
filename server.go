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
)

const uploadDir = "uploads"

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
	fmt.Println("Successfully uploaded", tempDir+"/"+fileName)

	return tempDir, fileName, nil
}

// Error TODO
type Error struct {
	Name    string
	Details string
}

// Task TODO
type Task struct {
	Name         string
	PassingBuild bool
	Errors       []Error
	Tests        []bool
}

// Response TODO
type Response struct {
	PageTitle string
	Tasks     []Task
	Errors    []Error
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	response := Response{"File name error", nil, nil}
	defer func() {
		tmpl, err := template.New("homework.html").Funcs(template.FuncMap{"escapeNewLineHTML": escapeNewLineHTML}).ParseFiles("templates/homework.html")
		if err != nil {
			log.Fatal(err)
		}
		err = tmpl.Execute(w, response)
		if err != nil {
			log.Fatal(err)
		}
	}()

	tempDir, fileName, err := retrieveFile(r)
	if err != nil {
		response.Errors = append(response.Errors, Error{"Failed upload", err.Error()})
		return
	}
	response.PageTitle = fileName
	defer os.RemoveAll(tempDir)

	taskDirs, err := unzip(tempDir+"/"+fileName, tempDir)
	if err != nil {
		response.Errors = append(response.Errors, Error{"Failed extract", err.Error()})
		return
	}

	sort.Strings(taskDirs)

	for _, dir := range taskDirs {
		response.Tasks = append(response.Tasks, Task{filepath.Base(dir), true, nil, nil})
		output, err := build(dir)
		if err != nil {
			response.Tasks[len(response.Tasks)-1].PassingBuild = false
			response.Tasks[len(response.Tasks)-1].Errors =
				append(response.Tasks[len(response.Tasks)-1].Errors, Error{"Failed build", output})
		}
	}

}

func setupRoutes() {
	http.HandleFunc("/upload", uploadFile)
	http.ListenAndServe(":8080", nil)
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
