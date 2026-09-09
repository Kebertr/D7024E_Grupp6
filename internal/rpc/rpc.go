package rpc

import (
	"fmt"
	"net"
	"time"
)

func sendPing(fromAddress string, address string) bool {
	fromAddr, err := net.ResolveUDPAddr("udp", fromAddress)
	if err != nil {
		return false
	}

	fullAddress, err := net.ResolveUDPAddr("udp", address)
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
	err = resp.SetDeadline(time.Now().Add(2 * time.Second))
	if err != nil {
		return false
	}
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

func confirmPing(conn net.PacketConn) bool {
	err := conn.SetDeadline(time.Now().Add(2 * time.Second))
	if err != nil {
		return false
	}
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
