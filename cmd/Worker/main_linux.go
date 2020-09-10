// Worker unix OSes

// +build linux

package main

import (
	"common"
	"flag"
	"sync"
	"worker"
)

func main() {
	var debugFlag = flag.Bool("debug", false, "Enable debug logging.")
	var server = flag.String("server", "localhost", "Server to communicate with.")
	flag.Parse()
	worker.DebugLog = *debugFlag

	common.SetupCloseHandler()

	var wg sync.WaitGroup

	wg.Add(1)
	go worker.RunHelloProtocol(*server, &wg)

	wg.Add(1)
	go worker.RunHeartbeatListener(&wg)

	wg.Add(1)
	go worker.RunWorkerListener(&wg)

	wg.Wait()
}
