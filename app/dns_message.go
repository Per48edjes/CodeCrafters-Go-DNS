package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// NewDNSHeader creates a new DNS header with the given options
func NewDNSHeader(opts DNSHeaderOptions) (*DNSHeader, error) {
	if err := validateHeaderOptions(opts); err != nil {
		return nil, err
	}
	header := DNSHeader{ID: opts.ID, QDCount: opts.QDCount, ANCount: opts.ANCount, NSCount: opts.NSCount, ARCount: opts.ARCount}
	header.Flags = opts.QR<<QRShift | opts.OpCode<<OpCodeShift | opts.AA<<AAShift | opts.TC<<TCShift | opts.RD<<RDShift | opts.RA<<RAShift | opts.Z<<ZShift | opts.RCode<<RCodeShift
	return &header, nil
}

// NewDNSQuestion creates a new DNS question section with the given options

func NewDNSQuestion(opts DNSQuestionOptions) (*DNSQuestion, error) {
	questionLabels := StringToLabels(opts.Question)
	question := DNSQuestion{
		Name:  questionLabels,
		Type:  opts.Type,
		Class: opts.Class,
	}
	return &question, nil
}

// NewDNSAnswer creates a new DNS answer section with the given resource records
func NewDNSAnswer(opts DNSAnswerOptions) (*DNSAnswer, error) {
	var answer DNSAnswer
	for _, record := range opts.ResourceRecords {
		question := StringToLabels(record.Name)
		data, err := IPToBytes(record.Data, record.Length)
		if err != nil {
			return nil, err
		}
		answer.ResourceRecords = append(answer.ResourceRecords, ResourceRecord{
			Name:   question,
			Type:   record.Type,
			Class:  record.Class,
			TTL:    record.TTL,
			Length: record.Length,
			Data:   data,
		})
	}
	return &answer, nil
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

// Serialize the DNS answer into a byte slice
func (answer *DNSAnswer) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	for _, record := range answer.ResourceRecords {
		for _, label := range record.Name {
			buf.WriteByte(label.Length)
			buf.WriteString(label.Content)
		}
		buf.WriteByte(0) // Null-terminate the sequence of labels
		err := binary.Write(buf, binary.BigEndian, record.Type)
		if err != nil {
			return nil, err
		}
		err = binary.Write(buf, binary.BigEndian, record.Class)
		if err != nil {
			return nil, err
		}
		err = binary.Write(buf, binary.BigEndian, record.TTL)
		if err != nil {
			return nil, err
		}
		err = binary.Write(buf, binary.BigEndian, record.Length)
		if err != nil {
			return nil, err
		}
		buf.Write(record.Data)
	}
	return buf.Bytes(), nil
}

// Deserialize the DNS header from a 12-byte slice
func (header *DNSHeader) Decode(encoded []byte) error {
	expectedSize := DNSHeaderSize
	if len(encoded) != expectedSize {
		return fmt.Errorf("Expected %d bytes in header, got %d", expectedSize, len(encoded))
	}
	buf := bytes.NewReader(encoded)
	if err := binary.Read(buf, binary.BigEndian, header); err != nil {
		return err
	}
	return nil
}

// ModifyDNSHeader modifies an existing DNS header with the given options; if any modification fails, the original header is returned
func (header *DNSHeader) ModifyDNSHeader(modifications ...DNSHeaderModification) (*DNSHeader, error) {
	newHeader := *header
	for _, modification := range modifications {
		if err := modification(&newHeader); err != nil {
			return header, err
		}
	}
	return &newHeader, nil
}

// ModifyQR modifies the QR field of a DNS header
func ModifyQR(qr uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		validate := validateQR(qr)
		if err := validate(); err != nil {
			return err
		}
		header.Flags = header.Flags&^QRMask | qr<<QRShift
		return nil
	}
}

// ModifyOpCode modifies the OpCode field of a DNS header
func ModifyOpCode(opCode uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		validate := validateOpCode(opCode)
		if err := validate(); err != nil {
			return err
		}
		header.Flags = header.Flags&^OpCodeMask | opCode<<OpCodeShift
		return nil
	}
}

// ModifyAA modifies the AA field of a DNS header
func ModifyAA(aa uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		validate := validateAA(aa)
		if err := validate(); err != nil {
			return err
		}
		header.Flags = header.Flags&^AAMask | aa<<AAShift
		return nil
	}
}

// ModifyTC modifies the TC field of a DNS header
func ModifyTC(tc uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		validate := validateTC(tc)
		if err := validate(); err != nil {
			return err
		}
		header.Flags = header.Flags&^TCMask | tc<<TCShift
		return nil
	}
}

// ModifyRD modifies the RD field of a DNS header
func ModifyRD(rd uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		validate := validateRD(rd)
		if err := validate(); err != nil {
			return err
		}
		header.Flags = header.Flags&^RDMask | rd<<RDShift
		return nil
	}
}

// ModifyRA modifies the RA field of a DNS header
func ModifyRA(ra uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		validate := validateRA(ra)
		if err := validate(); err != nil {
			return err
		}
		header.Flags = header.Flags&^RAMask | ra<<RAShift
		return nil
	}
}

// ModifyZ modifies the Z field of a DNS header
func ModifyZ(z uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		validate := validateZ(z)
		if err := validate(); err != nil {
			return err
		}
		header.Flags = header.Flags&^ZMask | z<<ZShift
		return nil
	}
}

// ModifyRCode modifies the RCode field of a DNS header
func ModifyRCode(rCode uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		validate := validateRCode(rCode)
		if err := validate(); err != nil {
			return err
		}
		header.Flags = header.Flags&^RCodeMask | rCode<<RCodeShift
		return nil
	}
}
