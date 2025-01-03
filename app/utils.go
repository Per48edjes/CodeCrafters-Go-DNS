package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Convert a string into a list of DNSLabels
func StringToLabels(name string) ([]DNSLabel, error) {
	labels := []DNSLabel{}
	for _, label := range strings.Split(name, ".") {
		content := []byte(label)
		length := len(content)
		if length > 255 {
			return nil, fmt.Errorf("Label %s is too long", label)
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

// Convert a byte slice into a list of DNSLabels; consues all bytes in the input slice
func BytesToLabels(data []byte) ([]DNSLabel, error) {
	labels := []DNSLabel{}
	buf := bytes.NewReader(data)
	for buf.Len() > 0 {
		length, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		content := make([]byte, length)
		if _, err := buf.Read(content); err != nil {
			return nil, err
		}
		labels = append(labels, DNSLabel{Length: length, Content: content})
	}
	return labels, nil
}

// Convert an IP address into a byte slice; if invalid input, the function returns non-nil error
func IPToBytes(IPAddress string, dataLength uint16) ([]byte, error) {
	parts := strings.Split(IPAddress, ".")
	if len(parts) != int(dataLength) {
		return nil, fmt.Errorf("IP address length not %d", dataLength)
	}
	bytes := make([]byte, 4)
	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return nil, fmt.Errorf("Invalid IP address: %s", IPAddress)
		}
		bytes[i] = byte(num)
	}
	return bytes, nil
}

// readQName consumes bytes until a NULL byte or pointer is encountered to recover the uncompressed bytes of a DNS name
// - If a NULL byte is encountered, it is included in the result.
// - If a pointer is encountered, it recursively resolves and appends the pointed data.
func readQName(buf *bytes.Reader) ([]byte, error) {
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
			pointedData, err := readQName(buf)          // Recursively resolve the pointer
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
