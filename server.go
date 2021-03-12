package main

import (
	"flag"
	"log"

	"github.com/luchev/dtf/master"
	"github.com/luchev/dtf/util"
	"github.com/luchev/dtf/worker"
)

const uploadDir = "uploads"

// Flags specifies the different args
type Flags struct {
	master bool
	worker bool
	port   int
}

func initFlags() Flags {
	var flags Flags
	flag.BoolVar(&flags.master, "master", false, "Start a master service")
	flag.BoolVar(&flags.worker, "worker", false, "Start a worker service")
	flag.IntVar(&flags.port, "port", 80, "Port to start the service on")
	flag.Parse()

	return flags
}

func main() {
	flags := initFlags()

	if flags.master {
		util.InitWorkspace(uploadDir)
		log.Println("Starting a Master service")
		master.SetupRoutes(flags.port)
	} else if flags.worker {
		util.InitWorkspace(uploadDir)
		worker.SetupRoutes(flags.port)
		log.Println("Starting a Worker service")
	} else {
		flag.Usage()
	}
}
