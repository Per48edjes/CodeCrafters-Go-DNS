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

// Reads from a UDP connection and processes the received data
func readAndProcess(conn *net.UDPConn, bytesBuffer []byte, isClient bool) (*DNSMessage, error) {
	var size int
	var source *net.UDPAddr
	var err error
	if isClient {
		size, err = conn.Read(bytesBuffer) // Client: pre-connected
		if err != nil {
			return nil, err
		}
	} else {
		size, source, err = conn.ReadFromUDP(bytesBuffer) // Server: listen mode
		if err != nil {
			return nil, err
		}
		fmt.Printf("Received %d bytes from %s\n", size, source)
	}
	buf := bytes.NewReader(bytesBuffer[:size])
	m := &DNSMessage{}
	if err = m.Decode(buf); err != nil {
		return nil, err
	}
	return m, nil
}
