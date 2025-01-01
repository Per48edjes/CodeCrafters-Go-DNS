package main

import (
	"bytes"
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

	b := make([]byte, 512)

eventLoop:
	for {
		size, source, err := udpConn.ReadFromUDP(b)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := string(b[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		buf := bytes.NewReader(b[:size])

		receivedMessage := &DNSMessage{}
		if err = receivedMessage.Decode(buf); err != nil {
			fmt.Println("Failed to decode received DNS message:", err)
			break
		}

		// Modify the header to reflect that this is a response
		receivedMessage.Header, err = receivedMessage.Header.ModifyDNSHeader(
			ModifyANCount(receivedMessage.Header.QDCount),
			ModifyQR(1),
			ModifyAA(0),
			ModifyTC(0),
			ModifyRA(0),
			ModifyZ(0),
		)
		if err != nil {
			fmt.Println("Failed to modify DNS header:", err)
			break eventLoop
		}

		// Modify the questions to reflect the response
		for i, question := range receivedMessage.Questions {
			var name string
			var answer *DNSAnswer
			name, err = LabelsToString(question.Name)
			if err != nil {
				fmt.Println("Failed to convert labels to string:", err)
				break eventLoop
			}
			question, err = question.ModifyDNSQuestion(ModifyQType(1), ModifyClass(1))
			if err != nil {
				fmt.Println("Failed to modify DNS Questions:", err)
				break eventLoop
			}
			answer, err = NewDNSAnswer([]ResourceRecordOptions{{
				Name:   name,
				Type:   1,
				Class:  1,
				TTL:    60,
				Length: 4,
				Data:   "8.8.8.8",
			}})
			if err != nil {
				fmt.Println("Failed to create new DNS Answer:", err)
				break eventLoop
			}
			receivedMessage.Questions[i] = question
			receivedMessage.Answers = append(receivedMessage.Answers, answer)
		}

		response, err := receivedMessage.Encode()
		if err != nil {
			fmt.Println("Failed to encode response message:", err)
			break eventLoop
		}

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
