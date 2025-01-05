package main

import (
	"bytes"
	"fmt"
	"net"
)

func main() {
	// Establish UDP connection with upstream client
	clientAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	clientConn, err := net.ListenUDP("udp", clientAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer clientConn.Close()

	// Establish connection with downstream DNS server
	resolverAddr, err := parseResolverFlag()
	if err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		return
	}

	resolverConn, err := net.ListenUDP("udp", resolverAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer resolverConn.Close()

	b := make([]byte, 512)

eventLoop:
	for {
		size, source, err := clientConn.ReadFromUDP(b)
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

		// Split up received message into individual requests to forward
		requestMessages := receivedMessage.SplitDNSMessage()
		for _, requestMessage := range requestMessages {
			var request []byte
			request, err = requestMessage.Encode()
			if err != nil {
				fmt.Println("Failed to encode forward-ed request:", err)
				break eventLoop
			}

			_, err = clientConn.WriteToUDP(request, resolverAddr)
			if err != nil {
				fmt.Println("Failed to send response:", err)
			}
		}

		// Modify the header to reflect that this is a response
		receivedMessage.Header, err = receivedMessage.Header.ModifyDNSHeader(
			ModifyANCount(receivedMessage.Header.QDCount), // Message contains answers in equal number to questions
			ModifyQR(1), // Message is now a response
			ModifyAA(0),
			ModifyTC(0),
			ModifyRA(0),
			ModifyZ(0),
		)
		if err != nil {
			fmt.Println("Failed to modify DNS header:", err)
			break eventLoop
		}

		// NOTE: Everything below is from prior implementations

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

		_, err = clientConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
