package master

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	lg "github.com/luchev/dtf/logging"
	st "github.com/luchev/dtf/structs"
	"github.com/luchev/dtf/util"
)

var workers = make(map[string]struct{}, 0)

func SetupRoutes(port int) {
	log.Printf("Initializing Master server routes")
	log.Printf("Master service started on http://127.0.0.1:%d", port)

	router := mux.NewRouter()
	router.HandleFunc("/test", handleMasterTest)
	router.HandleFunc("/", handleMasterIndex)
	router.HandleFunc("/add_node", handleMasterAddNode)
	router.HandleFunc("/add_node_receiver", handleMasterAddNodeReceiver)
	router.HandleFunc("/status", handleMasterStatus)
	srv := http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("127.0.0.1:%d", port),
		WriteTimeout: 150 * time.Second,
		ReadTimeout:  300 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func handleMasterTest(w http.ResponseWriter, r *http.Request) {
	response := st.Response{PageTitle: "File name error", Tasks: nil, Errors: nil}

	defer func() {
		tmpl, _ := template.New("result.html").
			Funcs(template.FuncMap{"escapeNewLineHTML": util.EscapeNewLineHTML}).
			ParseFiles("templates/result.html")
		tmpl.Execute(w, response)
	}()

	// Get file from form
	r.ParseMultipartForm(10 << 20) // 10 MB files
	tempDir, fileName, err := util.RetrieveFile(r, "codeZip")
	if err != nil {
		response.Errors = append(response.Errors, st.Error{Name: "Failed upload", Details: err.Error()})
		return
	}
	response.PageTitle = fileName
	defer os.RemoveAll(tempDir)

	// Extract file
	localArchiveFilePath := tempDir + "/" + fileName
	sourceFiles, err := util.Unzip(localArchiveFilePath, tempDir)
	if err != nil {
		response.Errors = append(response.Errors, st.Error{Name: "Failed extract", Details: err.Error()})
		return
	}
	testNames := util.GetTestNamesFromFiles(tempDir, sourceFiles)

	// Build
	stderr, err := util.Build(tempDir)
	response.Tasks = append(response.Tasks, st.Task{Name: fileName, PassingBuild: true, BuildMessage: stderr, Errors: nil, Tests: nil})
	if err != nil {
		response.Tasks[0].PassingBuild = false
		response.Tasks[0].Errors = append(response.Tasks[0].Errors, st.Error{Name: "Failed build", Details: stderr})
		return
	}

	activeWorkers := util.GetActiveWorkers(workers)
	// Run tests on master
	if len(activeWorkers) == 0 {
		response.Errors = append(response.Errors, st.Error{Name: "No active workers, falling back to using Master", Details: "Go to /add_node to add workers"})
		log.Printf("Running tests for %s[%dm%s%s[%dm\n", lg.Escape, lg.Underline, tempDir, lg.Escape, lg.Reset)
		response.Tasks[0].Tests = util.RunTests(testNames, tempDir)
		return
	}

	// Run tests on workers
	var writeMutex sync.Mutex
	var wg sync.WaitGroup
	for index, chunk := range util.SplitChunks(testNames, len(activeWorkers)) {
		wg.Add(1)
		go func(index int, chunk []string) {
			worker := activeWorkers[index]
			log.Printf("Running %s for %s[%dm%s%s[%dm on %s[%dm%s%s[%dm\n",
				chunk, lg.Escape, lg.Underline, tempDir, lg.Escape, lg.Reset, lg.Escape, lg.Bold, worker, lg.Escape, lg.Reset)

			results, err := util.RunTestsRemotely(chunk, localArchiveFilePath, worker)
			writeMutex.Lock()
			if err != nil {
				response.Tasks[0].Errors = append(response.Tasks[0].Errors,
					st.Error{Name: fmt.Sprintf("Failed to run tests on %s", worker), Details: err.Error()})
			} else {
				response.Tasks[0].Tests = append(response.Tasks[0].Tests, results...)
			}
			writeMutex.Unlock()
			wg.Done()
		}(index, chunk)
	}
	wg.Wait()
}

func handleMasterIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.New("index.html").ParseFiles("templates/index.html")
	tmpl.Execute(w, r)
}

func handleMasterAddNode(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.New("add_node.html").ParseFiles("templates/add_node.html")
	tmpl.Execute(w, r)
}

func handleMasterAddNodeReceiver(w http.ResponseWriter, r *http.Request) {
	defer func() {
		tmpl, _ := template.New("status.html").
			Funcs(template.FuncMap{"escapeNewLineHTML": util.EscapeNewLineHTML}).
			ParseFiles("templates/status.html")
		tmpl.Execute(w, util.PingWorkers(workers))
	}()

	r.ParseForm()

	remote := r.PostFormValue("remote")
	workers[remote] = struct{}{}
}

func handleMasterStatus(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.New("status.html").
		Funcs(template.FuncMap{"escapeNewLineHTML": util.EscapeNewLineHTML}).
		ParseFiles("templates/status.html")
	tmpl.Execute(w, util.PingWorkers(workers))
}
