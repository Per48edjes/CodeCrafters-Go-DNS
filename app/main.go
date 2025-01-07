package main

import (
	"bytes"
	"fmt"
	"net"
)

func main() {
	// Establish UDP connection with upstream client
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	clientConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to client address:", err)
		return
	}
	defer clientConn.Close()

	// Establish UDP connection with downstream DNS server
	resolverAddr, err := parseResolverFlag()
	if err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		return
	}

eventLoop:
	for {
		// Read and process client message
		clientBytes := make([]byte, 512)
		size, source, err := clientConn.ReadFromUDP(clientBytes)
		if err != nil {
			fmt.Println("Failed to read client message:", err)
			break eventLoop
		}
		fmt.Printf("Received %d bytes from client at %s: %v\n", size, source, clientBytes[:size])
		buf := bytes.NewReader(clientBytes[:size])
		clientMessage := &DNSMessage{}
		if err = clientMessage.Decode(buf); err != nil {
			fmt.Println("Failed to process client message:", err)
		}
		if err != nil {
			fmt.Println("Failed to read and process client message:", err)
			break eventLoop
		}

		// Split up received message into individual requests to forward to downstream resolver
		requestMessages := clientMessage.SplitDNSMessage()
		downstreamResponses, err := DNSServerHandler(resolverAddr, requestMessages)
		if err != nil {
			fmt.Println("Failed to forward client requests to downstream server:", err)
			break eventLoop
		}

		// Modify the client response questions and populate client response answers
		var answerCount uint16
		for i, question := range clientMessage.Questions {
			question, err = question.ModifyDNSQuestion(ModifyQType(1), ModifyClass(1))
			if err != nil {
				fmt.Println("Failed to modify DNS Questions:", err)
				break eventLoop
			}
			clientMessage.Questions[i] = question
			if answers := downstreamResponses[i].Answers; len(answers) > 0 {
				clientMessage.Answers = append(clientMessage.Answers, answers[0])
				answerCount++
			}
		}

		// Modify the client response header
		clientMessage.Header, err = clientMessage.Header.ModifyDNSHeader(
			ModifyANCount(answerCount), // Update answer count
			ModifyQR(1),                // Mark message as a response
			ModifyAA(0),
			ModifyTC(0),
			ModifyRA(0),
			ModifyZ(0),
		)
		if err != nil {
			fmt.Println("Failed to modify DNS header:", err)
			break eventLoop
		}

		response, err := clientMessage.Encode()
		if err != nil {
			fmt.Println("Failed to encode client response message:", err)
			break eventLoop
		}

		_, err = clientConn.WriteToUDP(response, source)
		fmt.Printf("Response sent to client at %s: %v", source, response)
		if err != nil {
			fmt.Println("Failed to send client response:", err)
		}
	}
}
