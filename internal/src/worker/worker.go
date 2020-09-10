package worker

import (
	"bytes"
	"context"
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

type worker struct {
}

func (*worker) Heartbeat(ctx context.Context, request *pbMessages.Ping) (*pbMessages.Pong, error) {
	ver := request.GetVersion()
	if ver == 1 {

	}

	response := &pbMessages.Pong{
		Version: 1,
		Name:    "name",
	}
	return response, nil
}

func (*worker) Work(ctx context.Context, request *pbMessages.WorkRequest) (*pbMessages.WorkResponse, error) {
	command := request.GetCommand()
	args := request.GetCommandArgs()
	fmt.Println(command)

	response := &pbMessages.WorkResponse{
		Version: 1,
		Output:  execute(command, args),
	}
	return response, nil
}

// Get preferred outbound ip of this machine
func GetOutboundIP(server string) string {
	dst := fmt.Sprintf("%s:80", server)
	conn, err := net.Dial("udp", dst)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func RunHelloProtocol(server string, wg *sync.WaitGroup) {
	for true {
		opts := grpc.WithInsecure()
		constr := fmt.Sprintf("%s:50050", server)
		cc, err := grpc.Dial(constr, opts)
		if err != nil {
			log.Fatal(err)
		}
		localAddr := GetOutboundIP(server)
		defer cc.Close()

		networkclient := pbMessages.NewHelloServiceClient(cc)
		hreq := &pbMessages.HelloRequest{Version: 1, Ip: localAddr}
		if DebugLog {
			fmt.Printf("HelloRequest => | ")
		}
		_, err = networkclient.Hello(context.Background(), hreq)
		if err != nil {
			fmt.Println(err)
		} else {
			if DebugLog {
				fmt.Printf("<= HelloResponse\n")
			}
		}
		time.Sleep(20 * time.Second)
	}
}

func RunHeartbeatListener(wg *sync.WaitGroup) {
	address := "0.0.0.0:50051"
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Error %v", err)
	}
	fmt.Printf("Herd worker is listening on %v ...\n", address)

	s := grpc.NewServer()
	pbMessages.RegisterHeartbeatServiceServer(s, &worker{})

	s.Serve(lis)
}

func RunWorkerListener(wg *sync.WaitGroup) {
	address := "0.0.0.0:50052"
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Error %v", err)
	}
	fmt.Printf("Herd worker is listening on %v ...\n", address)

	s := grpc.NewServer()
	pbMessages.RegisterWorkServiceServer(s, &worker{})

	s.Serve(lis)
}

func execute(cmdstr string, args string) string {

	cmd := exec.Command(cmdstr)
	//cmd.Stdin = strings.NewReader("some input")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return out.String()
}
