package main

import (
	"commander"
	"common"
	"flag"
	"fmt"
	"sync"
)

func main() {
	var debugFlag = flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()
	commander.DebugLog = *debugFlag

	fmt.Println("Firing up the herd commander...")

	common.SetupCloseHandler()

	commander.Workers = make(commander.WorkerMap)

	var wg sync.WaitGroup

	wg.Add(1)
	go commander.StartHelloListener(&wg)

	wg.Add(1)
	go commander.RunHearbeat(&wg)

	wg.Add(1)
	go commander.RunWorkSender(&wg)

	commander.Commands = append(commander.Commands, common.Command{"ls", "-la"})

	wg.Wait()

}
