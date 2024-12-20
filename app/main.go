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
		receivedHeader := &DNSHeader{}
		if err := receivedHeader.Decode(buf[:DNSHeaderSize]); err != nil {
			fmt.Println("Failed to create DNS header:", err)
			break
		}
		var rCodeMod DNSHeaderModification
		if receivedHeader.Flags&OpCodeMask == 0 {
			rCodeMod = ModifyRCode(0)
		} else {
			rCodeMod = ModifyRCode(4)
		}
		testHeader, err := receivedHeader.ModifyDNSHeader(ModifyANCount(1), ModifyQR(1), ModifyAA(0), ModifyTC(0), ModifyRA(0), ModifyZ(0), rCodeMod)
		if err != nil {
			fmt.Println("Failed to modify DNS header:", err)
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
		testAnswer, err := NewDNSAnswer(DNSAnswerOptions{[]ResourceRecordOptions{{
			Name:   "codecrafters.io",
			Type:   1,
			Class:  1,
			TTL:    60,
			Length: 4,
			Data:   "8.8.8.8",
		}}})
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
