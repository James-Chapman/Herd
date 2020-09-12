package commander

import (
	"common"
	"context"
	"fmt"
	"log"
	"net"
	"pbMessages"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type WorkerMap map[string]int

func (wm WorkerMap) AddWorker(server string) {
	_, found := wm[server]
	if !found {
		WorkersMtx.Lock()
		wm[server] = 0
		WorkersMtx.Unlock()
	}
}

func (wm WorkerMap) AddNetError(server string) {
	_, found := wm[server]
	if found {
		WorkersMtx.Lock()
		wm[server] += 1
		WorkersMtx.Unlock()
	}
}

func (wm WorkerMap) ResetNetError(server string) {
	_, found := wm[server]
	if found {
		WorkersMtx.Lock()
		wm[server] = 0
		WorkersMtx.Unlock()
	}
}

func (wm WorkerMap) GetNetError(server string) int {
	_, found := wm[server]
	if found {
		return wm[server]
	}
	return -1
}

var (
	DebugLog   bool
	Commands   []common.Command
	Workers    WorkerMap
	WorkersMtx sync.Mutex
)

const heartbeatVersion = 1
const workVersion = 1

type commander struct {
}

// This function implements the Hello interface
func (*commander) Hello(ctx context.Context, request *pbMessages.HelloRequest) (*pbMessages.HelloResponse, error) {
	ver := request.GetVersion()
	if ver == 1 {
		Workers.AddWorker(request.GetIp())
	}
	if DebugLog {
		fmt.Printf("Received Hello message from %s [%s]\n", request.GetIp(), request.GetFqdn())
	}

	response := &pbMessages.HelloResponse{
		Version: 1,
	}
	return response, nil
}

// Start the HelloRequest listener
func StartHelloListener(wg *sync.WaitGroup) {
	address := "0.0.0.0:50050"
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Error %v", err)
	}
	fmt.Printf("Herd commander is listening for HelloRequest on %v ...\n", address)

	s := grpc.NewServer()
	pbMessages.RegisterHelloServiceServer(s, &commander{})

	s.Serve(lis)
}

// RunHearbeat responsible for sending ping (hearbeat) messages to workers
func RunHearbeat(wg *sync.WaitGroup) {
	for true {
		for host, _ := range Workers {
			sent := false
			for !sent {
				connStr := fmt.Sprintf("%s:50051", host)
				pMessage := &pbMessages.Ping{Version: heartbeatVersion}
				sent = SendHeartbeatMessage(connStr, pMessage)
			}
			if !sent {
				Workers.AddNetError(host)
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func SendHeartbeatMessage(connString string, message *pbMessages.Ping) bool {
	opts := grpc.WithInsecure()
	cc, err := grpc.Dial(connString, opts)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return false
	}
	defer cc.Close()

	networkclient := pbMessages.NewHeartbeatServiceClient(cc)

	if DebugLog {
		fmt.Printf("ping => | ")
	}
	response, err := networkclient.Heartbeat(context.Background(), message)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return false
	} else {
		if DebugLog {
			fmt.Printf("<= pong\n")
		}
		if response.GetVersion() != heartbeatVersion {
			fmt.Printf("Pong version [%d] doesn't match Ping version [%d].\n", response.GetVersion(), heartbeatVersion)
		}
	}
	cc.Close()
	return true
}

// RunWorkSender is responsible for sending out work units to workers
func RunWorkSender(wg *sync.WaitGroup) {
	for true {
		if len(Commands) > 0 {
			for host, errCount := range Workers {
				if errCount > 10 {
					continue
				}
				sent := false
				for !sent {
					connStr := fmt.Sprintf("%s:50052", host)
					pMessage := &pbMessages.WorkRequest{Version: workVersion, Command: "ls"}
					sent = SendWorkMessage(connStr, pMessage)
				}
			}
			time.Sleep(10 * time.Second)
		}
		time.Sleep(5 * time.Second)
	}

}

func SendWorkMessage(connString string, message *pbMessages.WorkRequest) bool {
	opts := grpc.WithInsecure()
	cc, err := grpc.Dial(connString, opts)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return false
	}
	defer cc.Close()

	networkclient := pbMessages.NewWorkServiceClient(cc)

	if DebugLog {
		fmt.Printf("workRequest.Command: %s => | ", message.GetCommand())
	}

	response, err := networkclient.Work(context.Background(), message)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		return false
	} else {
		if DebugLog {
			fmt.Printf("Received workResponse => [%v]\n", response.Output)
		}
		if response.GetVersion() != workVersion {
			fmt.Printf("WorkResponse version [%d] doesn't match WorkRequest version [%d].\n", response.GetVersion(), workVersion)
		}
	}
	cc.Close()
	return true
}
