package utils

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// UniswapV2Contracts contains the contract ABIs and bytecode for Uniswap V2
type UniswapV2Contracts struct {
	Factory UniswapContract `json:"factory"`
	Router  UniswapContract `json:"router"`
	WETH9   UniswapContract `json:"weth9"`
}

// UniswapContract represents a smart contract with ABI and bytecode
type UniswapContract struct {
	ABI      interface{} `json:"abi"`
	Bytecode string      `json:"bytecode"`
	Name     string      `json:"name"`
}

// UniswapV2DeploymentData contains the data needed for V2 deployment
type UniswapV2DeploymentData struct {
	Contracts UniswapV2Contracts   `json:"contracts"`
	Metadata  []DeploymentMetadata `json:"metadata"`
}

// DeploymentMetadata represents metadata for display in the UI
type DeploymentMetadata struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

// FetchUniswapV2Contracts fetches the contract ABIs and bytecode for Uniswap V2
func FetchUniswapV2Contracts() (*UniswapV2Contracts, error) {
	// Use hardcoded contract data for reliability
	// In production, these could be fetched from a reliable source or embedded

	factoryABI := []map[string]interface{}{
		{
			"inputs": []map[string]interface{}{
				{"internalType": "address", "name": "_feeToSetter", "type": "address"},
			},
			"stateMutability": "nonpayable",
			"type":            "constructor",
		},
		{
			"anonymous": false,
			"inputs": []map[string]interface{}{
				{"indexed": true, "internalType": "address", "name": "token0", "type": "address"},
				{"indexed": true, "internalType": "address", "name": "token1", "type": "address"},
				{"indexed": false, "internalType": "address", "name": "pair", "type": "address"},
				{"indexed": false, "internalType": "uint256", "name": "", "type": "uint256"},
			},
			"name": "PairCreated",
			"type": "event",
		},
		{
			"inputs": []map[string]interface{}{
				{"internalType": "address", "name": "tokenA", "type": "address"},
				{"internalType": "address", "name": "tokenB", "type": "address"},
			},
			"name": "createPair",
			"outputs": []map[string]interface{}{
				{"internalType": "address", "name": "pair", "type": "address"},
			},
			"stateMutability": "nonpayable",
			"type":            "function",
		},
	}

	routerABI := []map[string]interface{}{
		{
			"inputs": []map[string]interface{}{
				{"internalType": "address", "name": "_factory", "type": "address"},
				{"internalType": "address", "name": "_WETH", "type": "address"},
			},
			"stateMutability": "nonpayable",
			"type":            "constructor",
		},
		{
			"inputs": []map[string]interface{}{
				{"internalType": "address", "name": "tokenA", "type": "address"},
				{"internalType": "address", "name": "tokenB", "type": "address"},
				{"internalType": "uint256", "name": "amountADesired", "type": "uint256"},
				{"internalType": "uint256", "name": "amountBDesired", "type": "uint256"},
				{"internalType": "uint256", "name": "amountAMin", "type": "uint256"},
				{"internalType": "uint256", "name": "amountBMin", "type": "uint256"},
				{"internalType": "address", "name": "to", "type": "address"},
				{"internalType": "uint256", "name": "deadline", "type": "uint256"},
			},
			"name": "addLiquidity",
			"outputs": []map[string]interface{}{
				{"internalType": "uint256", "name": "amountA", "type": "uint256"},
				{"internalType": "uint256", "name": "amountB", "type": "uint256"},
				{"internalType": "uint256", "name": "liquidity", "type": "uint256"},
			},
			"stateMutability": "nonpayable",
			"type":            "function",
		},
	}

	wethABI := []map[string]interface{}{
		{
			"inputs":          []interface{}{},
			"stateMutability": "nonpayable",
			"type":            "constructor",
		},
		{
			"inputs":          []interface{}{},
			"name":            "deposit",
			"outputs":         []interface{}{},
			"stateMutability": "payable",
			"type":            "function",
		},
		{
			"inputs": []map[string]interface{}{
				{"internalType": "uint256", "name": "wad", "type": "uint256"},
			},
			"name":            "withdraw",
			"outputs":         []interface{}{},
			"stateMutability": "nonpayable",
			"type":            "function",
		},
	}

	// These are placeholder bytecodes - in production, use actual compiled bytecode
	// For now, we'll return empty bytecode and let the frontend handle compilation
	contracts := &UniswapV2Contracts{
		Factory: UniswapContract{
			ABI:      factoryABI,
			Bytecode: "", // Will be fetched or compiled separately
			Name:     "UniswapV2Factory",
		},
		Router: UniswapContract{
			ABI:      routerABI,
			Bytecode: "", // Will be fetched or compiled separately
			Name:     "UniswapV2Router02",
		},
		WETH9: UniswapContract{
			ABI:      wethABI,
			Bytecode: "", // Will be fetched or compiled separately
			Name:     "WETH9",
		},
	}

	return contracts, nil
}

// PrepareUniswapV2DeploymentData prepares all data needed for Uniswap V2 deployment
func PrepareUniswapV2DeploymentData(chainType, chainID string) (*UniswapV2DeploymentData, error) {
	contracts, err := FetchUniswapV2Contracts()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contracts: %w", err)
	}

	// Create metadata for the UI
	metadata := []DeploymentMetadata{
		{
			Title: "Deployment Type",
			Value: "Uniswap V2 Infrastructure",
		},
		{
			Title: "Contracts",
			Value: "Factory, Router02, WETH9",
		},
		{
			Title: "Chain",
			Value: fmt.Sprintf("%s (Chain ID: %s)", chainType, chainID),
		},
		{
			Title: "Deployment Order",
			Value: "1. WETH9 → 2. Factory → 3. Router02",
		},
		{
			Title: "Dependencies",
			Value: "Router requires Factory and WETH addresses",
		},
		{
			Title: "Gas Estimate",
			Value: "~2.5M gas total (approximate)",
		},
		{
			Title: "Post-Deployment",
			Value: "Automatically configures Uniswap settings",
		},
	}

	return &UniswapV2DeploymentData{
		Contracts: *contracts,
		Metadata:  metadata,
	}, nil
}

// FetchContractFromGitHub fetches contract source code from a GitHub repository
func FetchContractFromGitHub(url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch contract: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch contract: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

// ValidateUniswapVersion validates that the Uniswap version is supported
func ValidateUniswapVersion(version string) error {
	supportedVersions := []string{"v2"}

	for _, v := range supportedVersions {
		if version == v {
			return nil
		}
	}

	return fmt.Errorf("unsupported Uniswap version: %s. Supported versions: %v", version, supportedVersions)
}

// GetUniswapV2ContractURLs returns the GitHub URLs for Uniswap V2 contracts
func GetUniswapV2ContractURLs() map[string]string {
	return map[string]string{
		"factory": "https://raw.githubusercontent.com/Uniswap/v2-core/master/contracts/UniswapV2Factory.sol",
		"router":  "https://raw.githubusercontent.com/Uniswap/v2-periphery/master/contracts/UniswapV2Router02.sol",
		"weth9":   "https://raw.githubusercontent.com/Uniswap/v2-periphery/master/contracts/test/WETH9.sol",
	}
}

// DeployV2Uniswap handles the deployment logic for Uniswap V2
func DeployV2Uniswap(chainType, chainID string) (*UniswapV2DeploymentData, error) {
	// Validate chain type
	if chainType != "ethereum" {
		return nil, fmt.Errorf("Uniswap V2 deployment currently only supported on Ethereum")
	}

	// Prepare deployment data
	deploymentData, err := PrepareUniswapV2DeploymentData(chainType, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare deployment data: %w", err)
	}

	// Additional validation could be added here
	// - Check if contracts are already deployed
	// - Validate RPC connectivity
	// - Check gas prices

	return deploymentData, nil
}

// EstimateUniswapV2DeploymentGas estimates the gas cost for Uniswap V2 deployment
func EstimateUniswapV2DeploymentGas() map[string]uint64 {
	return map[string]uint64{
		"weth9":   500000,  // ~500k gas
		"factory": 1000000, // ~1M gas
		"router":  1000000, // ~1M gas
		"total":   2500000, // ~2.5M gas total
	}
}

// GenerateUniswapV2Metadata generates metadata for frontend display
func GenerateUniswapV2Metadata(chainType, chainID string, gasEstimates map[string]uint64) []DeploymentMetadata {
	return []DeploymentMetadata{
		{
			Title: "Deployment Type",
			Value: "Uniswap V2 Infrastructure",
		},
		{
			Title: "Contracts",
			Value: "WETH9, Factory, Router02",
		},
		{
			Title: "Network",
			Value: fmt.Sprintf("%s (Chain ID: %s)", chainType, chainID),
		},
		{
			Title: "Deployment Sequence",
			Value: "1. WETH9 → 2. Factory → 3. Router02",
		},
		{
			Title: "Total Gas Estimate",
			Value: fmt.Sprintf("%d gas (~%.1f ETH)", gasEstimates["total"], float64(gasEstimates["total"])*20e-9), // Assuming 20 gwei
		},
		{
			Title: "Contract Dependencies",
			Value: "Router depends on Factory and WETH addresses",
		},
		{
			Title: "Automatic Configuration",
			Value: "Uniswap settings will be auto-configured after deployment",
		},
	}
}
