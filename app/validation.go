package main

/*
This module contains the private validation logic for the various data comprising a DNS message.
*/

import "fmt"

type validator func() error

// validateHeaderOptions validates the DNS header options
func validateHeaderOptions(opts DNSHeaderOptions) error {
	validators := []validator{validateQR(opts.QR)}

	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

// validateQR validates the QR field of a DNS header
func validateQR(qr uint16) validator {
	return func() error {
		if qr > QRMax {
			return fmt.Errorf("invalid QR value: %d (must be 0 or 1)", qr)
		}
		return nil
	}
}

// validateOpCode validates the OpCode field of a DNS header
func validateOpCode(opCode uint16) validator {
	return func() error {
		if opCode > OpCodeMax {
			return fmt.Errorf("invalid OpCode value: %d (must be between 0 and 15)", opCode)
		}
		return nil
	}
}

// validateAA validates the AA field of a DNS header
func validateAA(aa uint16) validator {
	return func() error {
		if aa > AAMax {
			return fmt.Errorf("invalid AA value: %d (must be 0 or 1)", aa)
		}
		return nil
	}
}

// validateTC validates the TC field of a DNS header
func validateTC(tc uint16) validator {
	return func() error {
		if tc > TCMax {
			return fmt.Errorf("invalid TC value: %d (must be 0 or 1)", tc)
		}
		return nil
	}
}

// validateRD validates the RD field of a DNS header
func validateRD(rd uint16) validator {
	return func() error {
		if rd > RDMax {
			return fmt.Errorf("invalid RD value: %d (must be 0 or 1)", rd)
		}
		return nil
	}
}

// validateRA validates the RA field of a DNS header
func validateRA(ra uint16) validator {
	return func() error {
		if ra > RAMax {
			return fmt.Errorf("invalid RA value: %d (must be 0 or 1)", ra)
		}
		return nil
	}
}

// validateZ validates the Z field of a DNS header
func validateZ(z uint16) validator {
	return func() error {
		if z > ZMax {
			return fmt.Errorf("invalid Z value: %d (must be between 0 and 7)", z)
		}
		return nil
	}
}

// validateRCode validates the RCode field of a DNS header
func validateRCode(rCode uint16) validator {
	return func() error {
		if rCode > RCodeMax {
			return fmt.Errorf("invalid RCode value: %d (must be between 0 and 15)", rCode)
		}
		return nil
	}
}
