package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	localTestnetRPC = "http://localhost:8545"
)

func TestIsValidEthereumAddress(t *testing.T) {
	t.Run("ValidAddresses", func(t *testing.T) {
		validAddresses := []string{
			"0x1234567890123456789012345678901234567890",
			"0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", // example address
			"0x0000000000000000000000000000000000000000", // zero address
		}

		for _, addr := range validAddresses {
			assert.True(t, IsValidEthereumAddress(addr), "Address should be valid: %s", addr)
		}
	})

	t.Run("InvalidAddresses", func(t *testing.T) {
		invalidAddresses := []string{
			"",      // empty
			"0x123", // too short
			"0x12345678901234567890123456789012345678901", // too long
			"0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG",  // invalid hex chars
			"not-an-address", // completely invalid
		}

		for _, addr := range invalidAddresses {
			assert.False(t, IsValidEthereumAddress(addr), "Address should be invalid: %s", addr)
		}
	})
}
