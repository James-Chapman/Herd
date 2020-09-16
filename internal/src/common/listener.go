package common

import (
	"fmt"
	"log"
	"net"
)

// Get preferred outbound ip of this machine
func GetOutboundIP(server string) string {
	dst := fmt.Sprintf("%s:80", server)
	conn, err := net.Dial("udp", dst)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}
