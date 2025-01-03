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

		receivedMessage := &DNSMessage{}
		if err := receivedMessage.Decode(buf); err != nil {
			fmt.Println("Failed to decode received DNS message:", err)
			break
		}

		// Make changes to received message to construct response
		name, err := LabelsToString(receivedMessage.Questions.Name)
		if err != nil {
			fmt.Println("Failed to convert labels to string:", err)
			break
		}
		mods := []interface{}{
			ModifyANCount(1),
			ModifyQR(1),
			ModifyAA(0),
			ModifyTC(0),
			ModifyRA(0),
			ModifyZ(0),
			ModifyQType(1),
			ModifyClass(1),
			ModifyAnswer([]ResourceRecordOptions{{
				Name:   name,
				Type:   1,
				Class:  1,
				TTL:    60,
				Length: 4,
				Data:   "8.8.8.8",
			}}...),
		}

		responseMessage, err := receivedMessage.ModifyDNSMessage(mods...)
		if err != nil {
			fmt.Println("Failed to modify DNS received message to construct response:", err)
			break
		}

		response, err := responseMessage.Encode()
		if err != nil {
			fmt.Println("Failed to encode response message:", err)
			break
		}

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
