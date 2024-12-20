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

		// Create a test response parts
		testHeader := &DNSHeader{}
		if err := testHeader.Decode(buf[:12]); err != nil {
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
		testAnswer, err := NewDNSAnswer([]ResourceRecordOptions{{
			Name:   "codecrafters.io",
			Type:   1,
			Class:  1,
			TTL:    60,
			Length: 4,
			Data:   "8.8.8.8",
		}})
		if err != nil {
			fmt.Println("Failed to create DNS answer:", err)
			break
		}

		// Encode test response parts
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
		answer, err := testAnswer.Encode()
		if err != nil {
			fmt.Println("Failed to encode DNS answer:", err)
			break
		}

		// Splice together response parts
		response := append(append(header, question...), answer...)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
