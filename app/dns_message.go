package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
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
	questionLabels, err := StringToLabels(opts.Name)
	if err != nil {
		return nil, err
	}
	question := DNSQuestion{
		Name:  questionLabels,
		Type:  opts.Type,
		Class: opts.Class,
	}
	return &question, nil
}

// NewDNSAnswer creates a new DNS answer section with the given resource records
func NewDNSAnswer(opts []ResourceRecordOptions) (*DNSAnswer, error) {
	var answer DNSAnswer
	for _, record := range opts {
		labels, err := StringToLabels(record.Name)
		if err != nil {
			return nil, err
		}
		data := net.ParseIP(record.Data)
		if data == nil {
			return nil, fmt.Errorf("invalid IP address: %s", record.Data)
		}
		answer.ResourceRecords = append(answer.ResourceRecords, ResourceRecord{
			Name:   labels,
			Type:   record.Type,
			Class:  record.Class,
			TTL:    record.TTL,
			Length: record.Length,
			Data:   data.To4(),
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
		if label.Length == 0 {
			break
		}
		_, err := buf.Write(label.Content)
		if err != nil {
			return nil, err
		}
	}
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
			_, err := buf.Write(label.Content)
			if err != nil {
				return nil, err
			}
		}
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

// Serialize the DNS message into a byte slice to send to the client
func (message *DNSMessage) Encode() ([]byte, error) {
	header, err := message.Header.Encode()
	if err != nil {
		return nil, err
	}
	questions := new(bytes.Buffer)
	for _, question := range message.Questions {
		encodedQuestion, err := question.Encode()
		if err != nil {
			return nil, err
		}
		questions.Write(encodedQuestion)
	}
	answers := new(bytes.Buffer)
	for _, answer := range message.Answers {
		encodedAnswer, err := answer.Encode()
		if err != nil {
			return nil, err
		}
		answers.Write(encodedAnswer)
	}
	return append(header, append(questions.Bytes(), answers.Bytes()...)...), nil
}

// Deserialize the DNS header from a 12-byte slice
func (header *DNSHeader) Decode(buf *bytes.Reader) error {
	if err := binary.Read(buf, binary.BigEndian, header); err != nil {
		return err
	}
	return nil
}

// Deserialize the DNS question from the byte slice after the header in a query
func (question *DNSQuestion) Decode(buf *bytes.Reader) error {
	qNameBytes, err := ReadQName(buf)
	if err != nil {
		return err
	}
	qName, err := BytesToLabels(qNameBytes)
	if err != nil {
		return err
	}
	question.Name = qName
	if err := binary.Read(buf, binary.BigEndian, &question.Type); err != nil {
		return err
	}
	if err := binary.Read(buf, binary.BigEndian, &question.Class); err != nil {
		return err
	}
	return nil
}

// Deserialize the DNS message from a byte slice received from the client
func (message *DNSMessage) Decode(buf *bytes.Reader) error {
	// Parse header
	receivedHeader := &DNSHeader{}
	if err := receivedHeader.Decode(buf); err != nil {
		return err
	}
	// Parse questions
	receivedQuestions := make([]*DNSQuestion, receivedHeader.QDCount)
	for i := 0; i < int(receivedHeader.QDCount); i++ {
		receivedQuestion := &DNSQuestion{}
		if err := receivedQuestion.Decode(buf); err != nil {
			return err
		}
		receivedQuestions[i] = receivedQuestion
	}
	// Change header response code from query
	var rCodeMod DNSHeaderModification
	if receivedHeader.Flags&OpCodeMask == 0 {
		rCodeMod = ModifyRCode(0) // No Error
	} else {
		rCodeMod = ModifyRCode(4) // Not Implemented
	}
	receivedHeader, err := receivedHeader.ModifyDNSHeader(rCodeMod)
	if err != nil {
		return err
	}
	message.Header, message.Questions, message.Answers = receivedHeader, receivedQuestions, []*DNSAnswer{} // Empty answer section
	return nil
}

// ModifyDNSHeader modifies an existing DNS header with the given options; if any modification fails, the original header is returned
func (header *DNSHeader) ModifyDNSHeader(modifications ...DNSHeaderModification) (*DNSHeader, error) {
	newHeader := *header
	for _, mod := range modifications {
		if err := mod(&newHeader); err != nil {
			return header, err
		}
	}
	return &newHeader, nil
}

// ModifyDNSQuestion modifies an existing DNS question with the given options; if any modification fails, the original question is returned
func (question *DNSQuestion) ModifyDNSQuestion(modifications ...DNSQuestionModification) (*DNSQuestion, error) {
	newQuestion := *question
	for _, mod := range modifications {
		if err := mod(&newQuestion); err != nil {
			return question, err
		}
	}
	return &newQuestion, nil
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

// ModifyQDCount modifies the QDCount field of a DNS header
func ModifyQDCount(qdCount uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		header.QDCount = qdCount
		return nil
	}
}

// ModifyANCount modifies the ANCount field of a DNS header
func ModifyANCount(anCount uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		header.ANCount = anCount
		return nil
	}
}

// ModifyNSCount modifies the NSCount field of a DNS header
func ModifyNSCount(nsCount uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		header.NSCount = nsCount
		return nil
	}
}

// ModifyARCount modifies the ARCount field of a DNS header
func ModifyARCount(arCount uint16) DNSHeaderModification {
	return func(header *DNSHeader) error {
		header.ARCount = arCount
		return nil
	}
}

// ModifyQType modifies the Type field of a DNS question
func ModifyQType(qType uint16) DNSQuestionModification {
	return func(question *DNSQuestion) error {
		question.Type = qType
		return nil
	}
}

// ModifyClass modifies the Class field of a DNS question
func ModifyClass(qClass uint16) DNSQuestionModification {
	return func(question *DNSQuestion) error {
		question.Class = qClass
		return nil
	}
}
