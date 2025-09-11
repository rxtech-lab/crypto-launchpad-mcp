package utils

import (
	"github.com/ethereum/go-ethereum/common"
)

// IsValidEthereumAddress checks if an Ethereum address is valid
func IsValidEthereumAddress(address string) bool {
	return common.IsHexAddress(address)
}
