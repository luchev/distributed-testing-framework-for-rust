package worker

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	lg "github.com/luchev/dtf/logging"
	"github.com/luchev/dtf/util"
)

func SetupRoutes(port int) {
	log.Printf("Initializing Worker server routes")
	log.Printf("Worker service started on http://127.0.0.1:%d", port)

	router := mux.NewRouter()
	router.HandleFunc("/test", handleWorkerTest)
	router.HandleFunc("/ping", handleWorkerPing)
	srv := http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("127.0.0.1:%d", port),
		WriteTimeout: 150 * time.Second,
		ReadTimeout:  300 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func handleWorkerTest(w http.ResponseWriter, r *http.Request) {
	log.Printf("POST /test")

	result := ""
	defer func() {
		io.WriteString(w, result)
	}()

	r.ParseMultipartForm(10 << 20) // 10 MB files
	tempDir, fileName, err := util.RetrieveFile(r, "codeZip")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		result = err.Error()
		log.Printf("Error uploading file %s[%dm%s%s[%dm: %s",
			lg.Escape, lg.Underline, tempDir+"/"+fileName, lg.Escape, lg.Reset, err.Error())
		return
	}
	defer os.RemoveAll(tempDir)

	testNames := strings.Split(r.FormValue("testList"), ",")

	localArchiveFilePath := tempDir + "/" + fileName
	_, err = util.Unzip(localArchiveFilePath, tempDir)
	if err != nil {
		result = "Failed extract: " + err.Error()
		log.Printf("Error extracting file %s[%dm%s%s[%dm: %s",
			lg.Escape, lg.Underline, tempDir+"/"+fileName, lg.Escape, lg.Reset, err.Error())
		return
	}

	// Test
	log.Printf("Running tests for %s[%dm%s%s[%dm\n", lg.Escape, lg.Underline, tempDir, lg.Escape, lg.Reset)
	results := util.RunTests(testNames, tempDir)

	encoded, err := json.Marshal(results)
	if err != nil {
		result = "Failed to run tests: " + err.Error()
		log.Printf("Failed to run tests for %s[%dm%s%s[%dm: %s",
			lg.Escape, lg.Underline, tempDir+"/"+fileName, lg.Escape, lg.Reset, err.Error())
		return
	}
	w.Write(encoded)
}

func handleWorkerPing(w http.ResponseWriter, r *http.Request) {
	log.Printf("GET /ping")

	io.WriteString(w, "OK")
}
