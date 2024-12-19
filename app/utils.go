package main

import "strings"

// Convert a string into a list of DNSQestionLabels
func nameToLabels(name string) []DNSQuestionLabel {
	labels := []DNSQuestionLabel{}
	for _, label := range strings.Split(name, ".") {
		labels = append(labels, DNSQuestionLabel{Length: uint8(len(label)), Content: label})
	}
	return labels
}
