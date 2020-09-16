package commander

import (
	"bytes"
	"common"
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"pbMessages"
	"sync"
	"time"

	"google.golang.org/grpc"
)

var (
	DebugLog   bool
	Commands   []common.Job
	Workers    WorkerMap
	WorkersMtx sync.Mutex
)

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
			if Workers.GetStatus(host) == WORKER_ONLINE {
				sent := false
				retry := 0
				for retry < 5 {
					connStr := fmt.Sprintf("%s:50051", host)
					pMessage := &pbMessages.Ping{Name: "name 1"}
					sent = SendHeartbeatMessage(connStr, pMessage)
					if sent {
						retry = 5
					} else {
						time.Sleep(1 * time.Second)
					}
					retry += 1
				}
				if !sent {
					Workers.AddNetError(host)
					if Workers.GetNetErrors(host) > 10 {
						fmt.Printf("Setting %s to OFFLINE\n", host)
						Workers.SetStatus(host, WORKER_OFFLINE)
					}
				} else {
					if Workers.GetStatus(host) == WORKER_OFFLINE {
						fmt.Printf("Setting %s to ONLINE\n", host)
						Workers.SetStatus(host, WORKER_ONLINE)
						Workers.ResetNetError(host)
					}
				}
			}
			if Workers.GetStatus(host) == WORKER_OFFLINE {
				sent := false
				connStr := fmt.Sprintf("%s:50051", host)
				pMessage := &pbMessages.Ping{Name: "name 1"}
				sent = SendHeartbeatMessage(connStr, pMessage)
				if sent {
					fmt.Printf("Setting %s to ONLINE\n", host)
					Workers.SetStatus(host, WORKER_ONLINE)
					Workers.ResetNetError(host)
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func SendHeartbeatMessage(connString string, message *pbMessages.Ping) bool {
	opts := grpc.WithInsecure()
	cc, err := grpc.Dial(connString, opts)
	if err != nil {
		if DebugLog {
			log.Printf("gRPC dial error: %v\n", err)
		}
		return false
	}
	defer cc.Close()

	networkclient := pbMessages.NewHeartbeatServiceClient(cc)
	response, err := networkclient.Heartbeat(context.Background(), message)
	if err != nil {
		if DebugLog {
			log.Printf("SendHeartbeatMessage() failed: %v\n", err)
		}
		return false
	} else {
		if response != nil {
			if DebugLog {
				fmt.Printf("Sent 'Ping' to 'Heartbeat' service, received 'Pong'\n")
			}
		}
	}
	cc.Close()
	return true
}

// RunWorkSender is responsible for sending out work units to workers
func RunWorkSender(wg *sync.WaitGroup) {
	for true {
		if len(Commands) > 0 {
			for host, _ := range Workers {
				// For each host we know about
				if Workers.GetNetErrors(host) > 10 {
					continue
				}
				// if node is online, try and send
				if Workers.GetStatus(host) == WORKER_ONLINE {
					sent := false
					retry := 0
					for retry < 5 {
						connStr := fmt.Sprintf("%s:50052", host)
						// construct the job struct
						var job common.Job
						job.Command = "ls"
						job.Args = append(job.Args, "-l")
						job.Status = common.WAITING

						// serialise the struct into buffer
						var buffer bytes.Buffer
						enc := gob.NewEncoder(&buffer)
						err := enc.Encode(job)
						if err != nil {
							log.Println("encode error:", err)
						}

						// turn buffer into []byte for protocol buffers message
						jobdata := buffer.Bytes()

						//construct the message and send
						pMessage := &pbMessages.WorkRequest{JobID: 1, Job: jobdata}
						sent = SendWorkMessage(connStr, pMessage)
						if sent {
							retry = 5
						} else {
							Workers.AddNetError(host)
							time.Sleep(1 * time.Second)
						}
						retry += 1
					}
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
		if DebugLog {
			log.Printf("gRPC dial error: %v\n", err)
		}
		return false
	}
	defer cc.Close()

	networkclient := pbMessages.NewWorkServiceClient(cc)
	response, err := networkclient.Work(context.Background(), message)
	if err != nil {
		if DebugLog {
			log.Printf("SendWorkMessage() failed: %v\n", err)
		}
		return false
	} else {
		if response != nil {
			if DebugLog {
				fmt.Printf("Sent 'WorkRequest' to 'Work' service, received 'WorkResponse'\n")
				fmt.Printf("WorkResponse.Output:\n%v\n", response.Output)
			}
		}
	}
	cc.Close()
	return true
}
