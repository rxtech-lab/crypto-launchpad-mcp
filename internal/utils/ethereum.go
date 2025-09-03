package utils

import "github.com/ethereum/go-ethereum/common"

func IsValidEthereumAddress(address string) bool {
	return common.IsHexAddress(address)
}
