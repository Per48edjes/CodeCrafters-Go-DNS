package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// NewDNSMessage creates a new DNS message with the given options
func NewDNSMessage(headerOpts DNSHeaderOptions, questionOpts DNSQuestionOptions, answerOpts DNSAnswerOptions) (*DNSMessage, error) {
	header, err := NewDNSHeader(headerOpts)
	if err != nil {
		return nil, err
	}
	question, err := NewDNSQuestion(questionOpts)
	if err != nil {
		return nil, err
	}
	answer, err := NewDNSAnswer(answerOpts)
	if err != nil {
		return nil, err
	}
	return &DNSMessage{Header: header, Questions: question, Answers: answer}, nil
}

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
func NewDNSAnswer(opts DNSAnswerOptions) (*DNSAnswer, error) {
	var answer DNSAnswer
	for _, record := range opts.ResourceRecords {
		question, err := StringToLabels(record.Name)
		if err != nil {
			return nil, err
		}
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

// Serialize the DNS message into a byte slice
func (message *DNSMessage) Encode() ([]byte, error) {
	headerBytes, err := message.Header.Encode()
	if err != nil {
		return nil, err
	}
	questionBytes, err := message.Questions.Encode()
	if err != nil {
		return nil, err
	}
	answerBytes, err := message.Answers.Encode()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(headerBytes)
	buf.Write(questionBytes)
	buf.Write(answerBytes)
	return buf.Bytes(), nil
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
		_, err := buf.Write(label.Content)
		if err != nil {
			return nil, err
		}
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
			_, err := buf.Write(label.Content)
			if err != nil {
				return nil, err
			}
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

// Deserialize the DNS question from the byte slice after the header in a query
func (question *DNSQuestion) Decode(buf *bytes.Reader) error {
	qNameBytes, err := readQName(buf)
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

// Deserialize the DNS answer from the byte slice from a query; overwrites the existing header and question is messaege is not nil
func (message *DNSMessage) Decode(encoded []byte) error {
	header, questions := encoded[:DNSHeaderSize], encoded[DNSHeaderSize:]
	// Parse header
	buf, receivedHeader := bytes.NewReader(header), &DNSHeader{}
	if err := receivedHeader.Decode(header); err != nil {
		return err
	}
	// Parse questions
	buf = bytes.NewReader(questions)
	receivedQuestions := make([]*DNSQuestion, receivedHeader.QDCount)
	for i := uint16(0); i < receivedHeader.QDCount; i++ {
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
	message.Header, message.Questions, message.Answers = receivedHeader, receivedQuestions, []*DNSAnswer{} // Empty answer section
	message.ModifyDNSMessage(rCodeMod)
	return nil
}

// TODO: Modify to accomodoate multiple DNSQuestions and DNSAnswers
// ModifyDNSMessage modifies an existing DNS message with the given options; if any modification fails, the original message is returned
func (message *DNSMessage) ModifyDNSMessage(modifications ...interface{}) (*DNSMessage, error) {
	newMessage := *message
	for _, modification := range modifications {
		switch mod := modification.(type) {
		case DNSHeaderModification:
			if err := mod(newMessage.Header); err != nil {
				return message, err
			}
		case DNSQuestionModification:
			if err := mod(newMessage.Questions); err != nil {
				return message, err
			}
		case DNSAnswerModification:
			if err := mod(newMessage.Answers); err != nil {
				return message, err
			}
		default:
			return message, fmt.Errorf("Unsupported modification type")
		}
	}
	return &newMessage, nil
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

// ModifyAnswer modifies the answer section of a DNS message with the given resource records
func ModifyAnswer(rrOpts ...ResourceRecordOptions) DNSAnswerModification {
	return func(answer *DNSAnswer) error {
		var addedResourceRecords []ResourceRecord
		for _, rrOpt := range rrOpts {
			question, err := StringToLabels(rrOpt.Name)
			if err != nil {
				return err
			}
			data, err := IPToBytes(rrOpt.Data, rrOpt.Length)
			if err != nil {
				return err
			}
			addedResourceRecords = append(addedResourceRecords, ResourceRecord{
				Name:   question,
				Type:   rrOpt.Type,
				Class:  rrOpt.Class,
				TTL:    rrOpt.TTL,
				Length: rrOpt.Length,
				Data:   data,
			})
		}
		answer.ResourceRecords = addedResourceRecords // Overwrite existing records (if any)
		return nil
	}
}
