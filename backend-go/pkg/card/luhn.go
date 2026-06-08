// Package card provides card number generation and validation utilities.
// Architecture doc §6: PAN structure, Luhn check digit, test card generation.
package card

import (
	"fmt"
	"math/rand"
	"strings"
)

// LuhnCheckDigit computes the Luhn check digit for a partial PAN.
// The input is the PAN body (without check digit).
func LuhnCheckDigit(body string) int {
	sum := 0
	double := false

	for i := len(body) - 1; i >= 0; i-- {
		d := int(body[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}

	return (10 - (sum % 10)) % 10
}

// ValidateLuhn checks if a full PAN passes the Luhn check.
func ValidateLuhn(pan string) bool {
	if len(pan) < 2 {
		return false
	}
	body := pan[:len(pan)-1]
	expected := int(pan[len(pan)-1] - '0')
	return LuhnCheckDigit(body) == expected
}

// GenerateTestPAN generates a test PAN using the given BIN and total length.
// Architecture doc §6.2: test_bin + random identifier + Luhn check digit.
// Test BIN examples (for sandbox only — not routable on real networks):
//
//	Visa:        "400000" (16 digits)
//	Mastercard:  "520000" (16 digits)
func GenerateTestPAN(testBIN string, totalLength int) string {
	if len(testBIN) >= totalLength {
		return ""
	}

	bodyLength := totalLength - 1
	randomLen := bodyLength - len(testBIN)

	var sb strings.Builder
	sb.WriteString(testBIN)
	for i := 0; i < randomLen; i++ {
		sb.WriteByte('0' + byte(rand.Intn(10)))
	}
	body := sb.String()

	check := LuhnCheckDigit(body)
	return fmt.Sprintf("%s%d", body, check)
}

// MaskPAN returns the last 4 digits of a PAN for display.
func MaskPAN(pan string) string {
	if len(pan) < 4 {
		return "****"
	}
	return pan[len(pan)-4:]
}

// FormatPANLast4 returns "****1234" style masked PAN.
func FormatPANLast4(last4 string) string {
	if len(last4) == 0 {
		return "****"
	}
	return fmt.Sprintf("****%s", last4)
}

// TestBINs returns common test BINs by network.
func TestBINs() map[string]string {
	return map[string]string{
		"VISA":       "400000",
		"MASTERCARD": "520000",
	}
}
