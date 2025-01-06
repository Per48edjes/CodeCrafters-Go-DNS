package main

import (
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

	resolverConn, err := net.DialUDP("udp", nil, resolverAddr)
	if err != nil {
		fmt.Println("Failed to bind to resolver address: ", err)
		return
	}
	defer resolverConn.Close()

	// Set up buffers for upsteam client and downstream resolver messages
	clientBytes := make([]byte, 512)
	downstreamBytes := make([]byte, 512)

eventLoop:
	for {
		var clientMessage *DNSMessage
		clientMessage, err = readAndProcess(clientConn, clientBytes, false)
		if err != nil {
			fmt.Println("Failed to read and process client message:", err)
			break eventLoop
		}

		// Split up received message into individual requests to forward to downstream resolver
		requestMessages := clientMessage.SplitDNSMessage()
		downstreamResponses := make([]*DNSMessage, len(requestMessages))
		for i, requestMessage := range requestMessages {
			var request []byte
			request, err = requestMessage.Encode()
			if err != nil {
				fmt.Println("Failed to encode forwarded request:", err)
				break eventLoop
			}

			// Send request to downstream resolver
			_, err = resolverConn.Write(request)
			if err != nil {
				fmt.Println("Failed to send request to downstream resolver:", err)
			}

			// Capture response from downstream resolver
			var downstreamMessage *DNSMessage
			downstreamMessage, err = readAndProcess(resolverConn, downstreamBytes, true)
			if err != nil {
				fmt.Println("Failed to read and process downstream response:", err)
				break eventLoop
			}
			downstreamResponses[i] = downstreamMessage
		}

		// Modify the client response header
		clientMessage.Header, err = clientMessage.Header.ModifyDNSHeader(
			ModifyANCount(clientMessage.Header.QDCount), // Message contains answers in equal number to questions
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

		for i, question := range clientMessage.Questions {
			// Modify the question to reflect the response
			question, err = question.ModifyDNSQuestion(ModifyQType(1), ModifyClass(1))
			if err != nil {
				fmt.Println("Failed to modify DNS Questions:", err)
				break eventLoop
			}
			clientMessage.Questions[i] = question
			// Copy answer from downstream response
			fmt.Println("Downstream response:", downstreamResponses[i])
			answer := downstreamResponses[i].Answers[0]
			clientMessage.Answers = append(clientMessage.Answers, answer)
		}

		response, err := clientMessage.Encode()
		if err != nil {
			fmt.Println("Failed to encode client response message:", err)
			break eventLoop
		}

		_, err = clientConn.WriteToUDP(response, udpAddr)
		if err != nil {
			fmt.Println("Failed to send client response:", err)
		}
	}
}
