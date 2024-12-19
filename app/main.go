package main

import (
	"fmt"
	"net"
)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := string(buf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		// Create a test response
		testHeader, err := NewDNSHeader(DNSHeaderOptions{
			ID:      1234,
			QR:      1,
			QDCount: 1,
		})
		if err != nil {
			fmt.Println("Failed to create DNS header:", err)
			break
		}
		testQuestion, err := NewDNSQuestion(DNSQuestionOptions{
			Question: "codecrafters.io",
			Type:     1,
			Class:    1,
		})
		if err != nil {
			fmt.Println("Failed to create DNS question:", err)
			break
		}

		// Encode test responses
		header, err := testHeader.Encode()
		if err != nil {
			fmt.Println("Failed to encode DNS header:", err)
			break
		}
		question, err := testQuestion.Encode()
		if err != nil {
			fmt.Println("Failed to encode DNS question:", err)
			break
		}

		// Splice together response parts
		response := append(header, question...)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
