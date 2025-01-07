package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
)

// Convert a string into a list of DNSLabels
func StringToLabels(name string) ([]DNSLabel, error) {
	labels := []DNSLabel{}
	for _, label := range strings.Split(name, ".") {
		content := []byte(label)
		length := len(content)
		if length > 255 {
			return nil, fmt.Errorf("label %s is too long", label)
		}
		labels = append(labels, DNSLabel{Length: uint8(length), Content: content})
	}
	return labels, nil
}

// Convert a list of DNSLabels into a string
func LabelsToString(labels []DNSLabel) (string, error) {
	parts := []string{}
	for _, label := range labels {
		parts = append(parts, string(label.Content))
	}
	return strings.Join(parts, "."), nil
}

// Convert a byte slice into a list of DNSLabels (with a "Null" label last); consumes all bytes in the input slice
func BytesToLabels(data []byte) ([]DNSLabel, error) {
	labels := []DNSLabel{}
	buf := bytes.NewReader(data)
	for buf.Len() > 0 {
		length, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		content := make([]byte, length)
		if length > 0 {
			if _, err := buf.Read(content); err != nil {
				return nil, err
			}
		}
		labels = append(labels, DNSLabel{Length: length, Content: content})
	}
	return labels, nil
}

// ReadQName consumes bytes until a NULL byte or pointer is encountered to recover the uncompressed bytes of a DNS name
// - If a NULL byte is encountered, it is included in the result.
// - If a pointer is encountered, it recursively resolves and appends the pointed data.
func ReadQName(buf *bytes.Reader) ([]byte, error) {
	var result []byte
	for {
		// Read the next byte
		b, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		switch {
		// Handle NULL byte (0x00)
		case b == 0x00:
			result = append(result, b) // Include the NULL byte
			return result, nil
		// Handle pointer (first octect will be 0xC0-0xFF)
		case b >= 0xC0:
			next, err := buf.ReadByte()
			if err != nil {
				return nil, err
			}
			offset := uint16(b&0x3F)<<8 | uint16(next)  // Extract the offset from the pointer
			currentPos := buf.Size() - int64(buf.Len()) // Current position
			buf.Seek(int64(offset), io.SeekStart)       // Move to the pointer offset
			pointedData, err := ReadQName(buf)          // Recursively resolve the pointer
			if err != nil {
				return nil, err
			}
			result = append(result, pointedData...)
			buf.Seek(currentPos, io.SeekStart) // Move back to the original position
			return result, nil
		default:
			result = append(result, b)
		}
	}
}

// Captures input to --resolver flag
func parseResolverFlag() (*net.UDPAddr, error) {
	resolverFlag := flag.String("resolver", "", "The resolver address in the form ip:port")
	flag.Parse()
	if *resolverFlag == "" {
		return nil, fmt.Errorf("please provide a resolver address with --resolver flag")
	}
	resolverAddr, err := net.ResolveUDPAddr("udp", *resolverFlag)
	if err != nil {
		return nil, err
	}
	return resolverAddr, nil
}

// Breaks a DNSMessage containing potentially multiple questions into a slice of individual DNSMessages
//   - The input message must have an empty DNSAnswer, which is replicated across ouput messages.
func (m *DNSMessage) SplitDNSMessage() []*DNSMessage {
	messages := make([]*DNSMessage, m.Header.QDCount)
	for i := uint16(0); i < m.Header.QDCount; i++ {
		newMessage := DNSMessage{Header: &DNSHeader{}, Questions: []*DNSQuestion{m.Questions[i]}, Answers: m.Answers}
		*newMessage.Header = *m.Header
		newMessage.Header.ModifyDNSHeader(ModifyQDCount(1))
		messages[i] = &newMessage
	}
	return messages
}

// Handles responses from downstream server for given set of requestMessages
func DNSServerHandler(downstreamAddr *net.UDPAddr, requestMessages []*DNSMessage) ([]*DNSMessage, error) {
	var downstreamResponses []*DNSMessage
	for _, requestMessage := range requestMessages {
		// Dial DNS server via UDP
		resolverConn, err := net.DialUDP("udp", nil, downstreamAddr)
		if err != nil {
			return nil, err
		}
		defer resolverConn.Close()

		// Modify the client response header
		requestMessage.Header, err = requestMessage.Header.ModifyDNSHeader(
			ModifyQDCount(1), // Sending only singleton questions to downstream server
		)
		if err != nil {
			return nil, err
		}

		// Send request to downstream resolver
		request, err := requestMessage.Encode()
		if err != nil {
			return nil, err
		}
		_, err = resolverConn.Write(request)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Sent %d bytes to downstream server: %v\n", len(request), request)

		// Read and process downstream server message
		downstreamMessage := &DNSMessage{}
		downstreamBytes := make([]byte, 512)
		size, err := resolverConn.Read(downstreamBytes)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Received %d bytes from downstream server: %v\n", size, downstreamBytes[:size])
		buf := bytes.NewReader(downstreamBytes[:size])
		if err = downstreamMessage.Decode(buf); err != nil {
			return nil, err
		}
		downstreamResponses = append(downstreamResponses, downstreamMessage)
	}
	return downstreamResponses, nil
}
