package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Convert a string into a list of DNSQestionLabels
func StringToLabels(name string) []DNSLabel {
	labels := []DNSLabel{}
	for _, label := range strings.Split(name, ".") {
		labels = append(labels, DNSLabel{Length: uint8(len(label)), Content: label})
	}
	return labels
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
