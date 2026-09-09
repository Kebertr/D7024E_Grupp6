package rpc

import (
	"fmt"
	"net"
	"time"
)

func sendPing(fromAddress string, fromPort int, address string, port int) bool {
	fromAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(fromAddress, fmt.Sprint(fromPort)))
	if err != nil {
		return false
	}
	fullAddress, err := net.ResolveUDPAddr("udp", net.JoinHostPort(address, fmt.Sprint(port)))
	if err != nil {
		return false
	}
	resp, err := net.DialUDP("udp", fromAddr, fullAddress)
	if err != nil {
		return false
	}
	if err != nil {
		return false
	}
	defer resp.Close()
	time_start := time.Now()
	_, err = resp.Write([]byte("Test"))
	if err != nil {
		return false
	}

	ans, err := resp.Read(make([]byte, 16))
	if err != nil {
		return false
	}
	if ans != 8 {
		return false
	}
	time_end := time.Now()
	fmt.Printf("Ping time: %v\n", time_end.Sub(time_start))
	return true
}

func confirmPing(address string, port int) bool {
	fullAddress := net.JoinHostPort(address, fmt.Sprint(port))
	conn, err := net.ListenPacket("udp", fullAddress)
	if err != nil {
		return false
	}
	defer conn.Close()
	ans, sender, err := conn.ReadFrom(make([]byte, 16))
	if err != nil {
		return false
	}
	if ans != 4 {
		return false
	}

	_, err = conn.WriteTo([]byte("TestBack"), sender)
	if err != nil {
		return false
	}
	return true
}
