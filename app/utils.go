package main

import (
	"fmt"
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

// Convert an IP address into a byte slice; if invalide input, the function returns non-nil error
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
