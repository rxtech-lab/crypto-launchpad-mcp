package services

type ContractDeploymentWithContractCodeTransactionArgs struct {
	ContractName    string `validate:"required"`
	ConstructorArgs []any  // Constructor arguments can be empty
	ContractCode    string `validate:"required"`
	Receiver        string `validate:"omitempty,eth_addr"` // Optional receiver address
	Value           string `validate:"omitempty,number"`   // Optional value, defaults to "0"
	Title           string `validate:"required"`
	Description     string `validate:"required"`
}

type ContractDeploymentWithBytecodeAndAbiTransactionArgs struct {
	Abi             string `validate:"required"`
	Bytecode        string `validate:"required"`
	ConstructorArgs []any  `validate:"required"`
	Receiver        string `validate:"omitempty,eth_addr"` // Optional receiver address
	Value           string `validate:"omitempty,number"`   // Optional value, defaults to "0"
	Title           string `validate:"required"`
	Description     string `validate:"required"`
}

type GetTransactionDataArgs struct {
	ContractAddress string `validate:"required,eth_addr"`
	FunctionName    string `validate:"required"`
	FunctionArgs    []any  `validate:"required"`
	Abi             string `validate:"required"`
	Value           string `validate:"omitempty,number"` // Optional value, defaults to "0"
	Title           string `validate:"required"`
	Description     string `validate:"required"`
}
