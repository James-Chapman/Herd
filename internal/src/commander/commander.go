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

type WorkerMap map[string]bool

func (wm WorkerMap) AddWorker(server string) {
	_, found := wm[server]
	if !found {
		WorkersMtx.Lock()
		wm[server] = true
		WorkersMtx.Unlock()
	}
}

var (
	DebugLog   bool
	Commands   []common.Command
	Workers    WorkerMap
	WorkersMtx sync.Mutex
)

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

// RunHearbeat is a loop responsible for sending
func RunHearbeat(wg *sync.WaitGroup) {
	for true {
		for k, _ := range Workers {
			opts := grpc.WithInsecure()
			constr := fmt.Sprintf("%s:50051", k)
			cc, err := grpc.Dial(constr, opts)
			if err != nil {
				log.Fatal(err)
			}
			defer cc.Close()
			const version = 1

			networkclient := pbMessages.NewHeartbeatServiceClient(cc)
			ping := &pbMessages.Ping{Version: version}
			if DebugLog {
				fmt.Printf("ping => | ")
			}
			pong, err = networkclient.Heartbeat(context.Background(), ping)
			if err != nil {
				fmt.Println(err)
			} else {
				if DebugLog {
					fmt.Printf("<= pong\n")
				}
				if pong.GetVersion() != version {
					fmt.Printf("Pong version [%d] doesn't match ping version [%d].\n", pong.GetVersion(), version)
				}
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// RunWorkSender is a loop responsible for sending out work units to workers
func RunWorkSender(wg *sync.WaitGroup) {
	for true {
		if len(Commands) > 0 {
			for k, _ := range Workers {
				opts := grpc.WithInsecure()
				constr := fmt.Sprintf("%s:50052", k)
				cc, err := grpc.Dial(constr, opts)
				if err != nil {
					log.Fatal(err)
				}
				defer cc.Close()

				networkclient := pbMessages.NewWorkServiceClient(cc)
				workrequest := &pbMessages.WorkRequest{Version: 1, Command: "ls"}
				if DebugLog {
					fmt.Printf("Work(\"ls\") => | ")
				}
				workresp, _ := networkclient.Work(context.Background(), workrequest)
				fmt.Printf("Receive response => [%v]\n", workresp.Output)
				time.Sleep(10 * time.Second)
			}
		}
		time.Sleep(5 * time.Second)
	}

}
