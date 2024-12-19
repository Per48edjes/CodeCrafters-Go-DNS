package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Encoder interface {
	Encode() ([]byte, error)
}

type Decoder interface {
	Decode([]byte) error
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

// DNSQuestionLabels are encoded as <length><content>, where <length> is a single byte that specifies the length of the label, and <content> is the actual content of the label. The sequence of labels is terminated by a null byte (\x00).
type DNSQuestionLabel struct {
	Length  uint8
	Content string
}

// DNSQuestion represents a list of questions that the sender
type DNSQuestion struct {
	Name  []DNSQuestionLabel
	Type  uint16
	Class uint16
}

// DNSQuestionOptions represents the options for creating a new DNSQuestion
type DNSQuestionOptions struct {
	Question string
	Type     uint16
	Class    uint16
}

// NewDNSQuestion creates a new DNS question section with the given options
func NewDNSQuestion(opts DNSQuestionOptions) (*DNSQuestion, error) {
	questionLabels := nameToLabels(opts.Question)
	question := DNSQuestion{
		Name:  questionLabels,
		Type:  opts.Type,
		Class: opts.Class,
	}
	return &question, nil
}

// NewDNSHeader creates a new DNS header with the given options
func NewDNSHeader(opts DNSHeaderOptions) (*DNSHeader, error) {
	// Validate options that are fewer than 16-bits each
	if opts.QR > 1 {
		return nil, fmt.Errorf("invalid QR value: %d (must be 0 or 1)", opts.QR)
	}
	if opts.OpCode > 15 {
		return nil, fmt.Errorf("invalid OpCode value: %d (must be 0-15)", opts.OpCode)
	}
	if opts.AA > 1 {
		return nil, fmt.Errorf("invalid AA value: %d (must be 0 or 1)", opts.AA)
	}
	if opts.TC > 1 {
		return nil, fmt.Errorf("invalid TC value: %d (must be 0 or 1)", opts.TC)
	}
	if opts.RD > 1 {
		return nil, fmt.Errorf("invalid RD value: %d (must be 0 or 1)", opts.RD)
	}
	if opts.RA > 1 {
		return nil, fmt.Errorf("invalid RA value: %d (must be 0 or 1)", opts.RA)
	}
	if opts.Z > 7 {
		return nil, fmt.Errorf("invalid Z value: %d (must be 0-7)", opts.Z)
	}
	if opts.RCode > 15 {
		return nil, fmt.Errorf("invalid RCode value: %d (must be 0-15)", opts.RCode)
	}
	header := DNSHeader{ID: opts.ID, QDCount: opts.QDCount, ANCount: opts.ANCount, NSCount: opts.NSCount, ARCount: opts.ARCount}
	// Set the flags field
	header.Flags = opts.QR<<15 | opts.OpCode<<11 | opts.AA<<10 | opts.TC<<9 | opts.RD<<8 | opts.RA<<7 | opts.Z<<4 | opts.RCode
	return &header, nil
}

// Serialize the DNS header into a 12-byte slice
func (header *DNSHeader) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, header)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Serialize the DNS question into a byte slice
func (question *DNSQuestion) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	for _, label := range question.Name {
		buf.WriteByte(label.Length)
		buf.WriteString(label.Content)
	}
	buf.WriteByte(0) // Null-terminate the sequence of labels
	err := binary.Write(buf, binary.BigEndian, question.Type)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, question.Class)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}