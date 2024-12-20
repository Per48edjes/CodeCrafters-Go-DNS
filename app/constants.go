package main

const (
	// DNSHeaderSize is the size of a DNS header in bytes
	DNSHeaderSize = 12
	// QRMax is the maximum value for the QR field
	QRMax = 1
	// OpCodeMax is the maximum value for the OpCode field
	OpCodeMax = 15
	// AAMax is the maximum value for the AA field
	AAMax = 1
	// TCMax is the maximum value for the TC field
	TCMax = 1
	// RDMax is the maximum value for the RD field
	RDMax = 1
	// RAMax is the maximum value for the RA field
	RAMax = 1
	// ZMax is the maximum value for the Z field
	ZMax = 7
	// RCodeMax is the maximum value for the RCode field
	RCodeMax = 15
	// QRShift is the number of bits to shift the QR field
	QRShift = 15
	// OpCodeShift is the number of bits to shift the OpCode field
	OpCodeShift = 11
	// AAShift is the number of bits to shift the AA field
	AAShift = 10
	// TCShift is the number of bits to shift the TC field
	TCShift = 9
	// RDShift is the number of bits to shift the RD field
	RDShift = 8
	// RAShift is the number of bits to shift the RA field
	RAShift = 7
	// ZShift is the number of bits to shift the Z field
	ZShift = 4
	// RCodeShift is the number of bits to shift the RCode field
	RCodeShift = 0
	// QRMasks is the mask for the QR field
	QRMask = 1 << QRShift
	// OpCodeMask is the mask for the OpCode field
	OpCodeMask = 15 << OpCodeShift
	// AAMask is the mask for the AA field
	AAMask = 1 << AAShift
	// TCMask is the mask for the TC field
	TCMask = 1 << TCShift
	// RDMask is the mask for the RD field
	RDMask = 1 << RDShift
	// RAMask is the mask for the RA field
	RAMask = 1 << RAShift
	// ZMask is the mask for the Z field
	ZMask = 7 << ZShift
	// RCodeMask is the mask for the RCode field
	RCodeMask = 15 << RCodeShift
)
