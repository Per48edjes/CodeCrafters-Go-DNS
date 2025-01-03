package main

import "bytes"

/*
This module contains the interfaces and types for the DNS message.
*/

type encoder interface {
	Encode() ([]byte, error)
}

type decoder interface {
	Decode(*bytes.Reader) error
}

type Serializable interface {
	encoder
	decoder
}

type DNSMessage struct {
	Header    *DNSHeader
	Questions []*DNSQuestion
	Answers   []*DNSAnswer
}

type DNSModification interface {
	DNSHeaderModification | DNSQuestionModification | DNSAnswerModification
}

// DNSHeaderModifications can be passed to ModifyDNSHeader to optionally change the header fields
type DNSHeaderModification func(*DNSHeader) error

// DNSQuestionModifications can be passed to ModifyDNSQuestion to optionally change the question fields
type DNSQuestionModification func(*DNSQuestion) error

// DNSAnswerModifications can be passed to ModifyDNSAnswer to optionally change the answer fields
type DNSAnswerModification func(*DNSAnswer) error

// DNSHeaderOptions represents the options for creating a new DNS header
type DNSHeaderOptions struct {
	ID      uint16
	QR      uint16
	OpCode  uint16
	AA      uint16
	TC      uint16
	RD      uint16
	RA      uint16
	Z       uint16
	RCode   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

// DNSHeader represents a 12-byte DNS header
type DNSHeader struct {
	ID      uint16 // Identifier
	Flags   uint16 // Flags and OpCode
	QDCount uint16 // Number of questions
	ANCount uint16 // Number of answers
	NSCount uint16 // Number of authority records
	ARCount uint16 // Number of additional records
}

// DNSQuestionLabels are encoded as <length><content>, where <length> is a single byte that specifies the length of the label, and <content> is the actual content of the label. The sequence of labels is terminated by a null byte (\x00).
type DNSLabel struct {
	Length  uint8
	Content []byte
}

// DNSQuestionOptions represents the options for creating a new DNSQuestion
type DNSQuestionOptions struct {
	Name  string
	Type  uint16
	Class uint16
}

// DNSQuestion represents a list of questions that the client wants to ask the server
type DNSQuestion struct {
	Name  []DNSLabel
	Type  uint16
	Class uint16
}

// ResourceRecordOption represents the options for creating a new ResourceRecord
type ResourceRecordOptions struct {
	Name   string
	Type   uint16
	Class  uint16
	TTL    uint32
	Length uint16
	Data   string
}

// ResourceRecord represents a resource record in the answer section of a DNS message
type ResourceRecord struct {
	Name   []DNSLabel
	Type   uint16
	Class  uint16
	TTL    uint32
	Length uint16
	Data   []byte
}

// DNSAnswerOptions is a wrapper around a list of ResourceRecordOptions
type DNSAnswerOptions struct {
	ResourceRecords []ResourceRecordOptions
}

// DNSAnswer represents a list of resource records that the answer the questions sent by the client
type DNSAnswer struct {
	ResourceRecords []ResourceRecord
}
