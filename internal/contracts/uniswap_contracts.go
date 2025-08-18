package contracts

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed uniswap/WETH9.json
var weth9JSON []byte

//go:embed uniswap/UniswapV2Factory.json
var factoryJSON []byte

//go:embed uniswap/UniswapV2Router02.json
var routerJSON []byte

// ContractArtifact represents a compiled contract artifact
type ContractArtifact struct {
	ABI      interface{} `json:"abi"`
	Bytecode string      `json:"bytecode"`
}

// GetWETH9Artifact returns the WETH9 contract artifact
func GetWETH9Artifact() (*ContractArtifact, error) {
	var artifact ContractArtifact
	if err := json.Unmarshal(weth9JSON, &artifact); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WETH9 artifact: %w", err)
	}
	return &artifact, nil
}

// GetFactoryArtifact returns the UniswapV2Factory contract artifact
func GetFactoryArtifact() (*ContractArtifact, error) {
	var artifact ContractArtifact
	if err := json.Unmarshal(factoryJSON, &artifact); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Factory artifact: %w", err)
	}
	return &artifact, nil
}

// GetRouterArtifact returns the UniswapV2Router02 contract artifact
func GetRouterArtifact() (*ContractArtifact, error) {
	var artifact ContractArtifact
	if err := json.Unmarshal(routerJSON, &artifact); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Router artifact: %w", err)
	}
	return &artifact, nil
}

// GetContractArtifact returns a contract artifact by name
func GetContractArtifact(name string) (*ContractArtifact, error) {
	switch name {
	case "WETH9":
		return GetWETH9Artifact()
	case "Factory":
		return GetFactoryArtifact()
	case "Router":
		return GetRouterArtifact()
	default:
		return nil, fmt.Errorf("unknown contract: %s", name)
	}
}
