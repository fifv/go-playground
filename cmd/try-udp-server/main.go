package main

import (
	"net"
	"github.com/charmbracelet/log"
)

func main() {
	udpServer, err := net.ListenUDP("udp", &net.UDPAddr{IP: nil, Port: 3398})
	if err != nil {
		log.Errorf("Err %v", err)
		return
	}
	defer udpServer.Close()

	buf := make([]byte, 2048)
	for {
		n, remoteAddr, err := udpServer.ReadFromUDP(buf)
		if err != nil {
			log.Infof("Err %v", err)
			continue
		}
		log.Infof("UDP got: %v [%v] %v", remoteAddr, n, string(buf[:n]))
	}
}
