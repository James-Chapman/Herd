package worker

import (
	"bytes"
	"common"
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os/exec"
	"sync"
	"time"

	"pbMessages"

	"google.golang.org/grpc"
)

var (
	DebugLog bool
)

const helloVersion = 1

type worker struct {
}

func (*worker) Heartbeat(ctx context.Context, request *pbMessages.Ping) (*pbMessages.Pong, error) {
	name := request.GetName()
	response := &pbMessages.Pong{
		Name: name,
	}
	return response, nil
}

func (*worker) Work(ctx context.Context, request *pbMessages.WorkRequest) (*pbMessages.WorkResponse, error) {
	// job data is []byte, but needs to be bytes.buffer for deserialisation
	byteData := request.GetJob()
	// create bytes.buffer
	buffer := bytes.NewBuffer(byteData)

	// deserialise into job struct
	decoder := gob.NewDecoder(buffer)
	var job common.Job
	err := decoder.Decode(&job)
	if err != nil {
		fmt.Printf("gob decode error: %v", err)
	}
	response := &pbMessages.WorkResponse{
		Output: executeCmd(job.Command, job.Args),
	}
	return response, nil
}

func RunHelloProtocol(server string, wg *sync.WaitGroup) {
	for true {
		localAddr := common.GetOutboundIP(server)
		connStr := fmt.Sprintf("%s:50050", server)
		sent := false

		pMessage := &pbMessages.HelloRequest{Version: 1, Ip: localAddr}
		sent = SendHelloMessage(connStr, pMessage)
		if !sent {
			log.Println("Sending HelloRequest failed.")
		}

		time.Sleep(20 * time.Second)
	}
}

func SendHelloMessage(connString string, message *pbMessages.HelloRequest) bool {
	opts := grpc.WithInsecure()
	cc, err := grpc.Dial(connString, opts)
	if err != nil {
		log.Printf("gRPC dial error: %v\n", err)
		return false
	}
	defer cc.Close()

	networkclient := pbMessages.NewHelloServiceClient(cc)
	response, err := networkclient.Hello(context.Background(), message)
	if err != nil {
		log.Printf("SendHelloMessage() failed: %v\n", err)
		return false
	} else {
		if response != nil {
			if DebugLog {
				fmt.Printf("Sent 'HelloRequest' to 'Hello' service, received 'HelloResponse'\n")
			}
		}
	}
	cc.Close()
	return true
}

func StartHeartbeatListener(wg *sync.WaitGroup) {
	address := "0.0.0.0:50051"
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
	}
	fmt.Printf("Herd worker is listening on %v ...\n", address)

	s := grpc.NewServer()
	pbMessages.RegisterHeartbeatServiceServer(s, &worker{})

	s.Serve(lis)
}

func StartWorkerListener(wg *sync.WaitGroup) {
	address := "0.0.0.0:50052"
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
	}
	fmt.Printf("Herd worker is listening on %v ...\n", address)

	s := grpc.NewServer()
	pbMessages.RegisterWorkServiceServer(s, &worker{})

	s.Serve(lis)
}

func executeCmd(cmdstr string, args []string) string {

	cmd := exec.Command(cmdstr)
	if DebugLog {
		fmt.Printf("cmd string: %s\n", cmdstr)
	}
	//cmd.Stdin = strings.NewReader("some input")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Printf("CMD ERROR: %v\n", err)
	}

	return out.String()
}
