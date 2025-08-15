package utils

import (
	"testing"
)

const code = `
// SPDX-License-Identifier: MIT
// Compatible with OpenZeppelin Contracts ^5.4.0
pragma solidity ^0.8.21;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import {ERC20Permit} from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Permit.sol";

contract MyToken is ERC20, ERC20Permit {
	constructor() ERC20("MyToken", "MTK") ERC20Permit("MyToken") {}
}
`

func TestCompileSolidity(t *testing.T) {
	type args struct {
		code string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				code: code,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompileSolidity("0.8.27", tt.args.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileSolidity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("CompileSolidity() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("CompileSolidity() error = %v, wantErr %v", err, tt.wantErr)
				}
			}

			if len(got.Bytecode) == 0 {
				t.Errorf("Expect bytecode to be non-empty")
			}

			if len(got.Abi) == 0 {
				t.Errorf("Expect abi to be non-empty")
			}
		})
	}
}
