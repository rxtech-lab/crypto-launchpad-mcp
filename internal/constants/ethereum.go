package constants

import "math/big"

var MaxUint256 = func() *big.Int {
	val := new(big.Int)
	val.SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)
	return val
}()
