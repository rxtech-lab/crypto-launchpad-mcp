package utils

import (
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/rxtech-lab/launchpad-mcp/internal/contracts"
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
	// Use embedded real contract artifacts from official Uniswap sources

	// Get WETH9 artifact
	wethArtifact, err := contracts.GetWETH9Artifact()
	if err != nil {
		return nil, fmt.Errorf("failed to get WETH9 artifact: %w", err)
	}

	// Get Factory artifact
	factoryArtifact, err := contracts.GetFactoryArtifact()
	if err != nil {
		return nil, fmt.Errorf("failed to get Factory artifact: %w", err)
	}

	// Get Router artifact
	routerArtifact, err := contracts.GetRouterArtifact()
	if err != nil {
		return nil, fmt.Errorf("failed to get Router artifact: %w", err)
	}

	// Build contracts struct with real artifacts
	contractsData := &UniswapV2Contracts{
		Factory: UniswapContract{
			ABI:      factoryArtifact.ABI,
			Bytecode: factoryArtifact.Bytecode,
			Name:     "UniswapV2Factory",
		},
		Router: UniswapContract{
			ABI:      routerArtifact.ABI,
			Bytecode: routerArtifact.Bytecode,
			Name:     "UniswapV2Router02",
		},
		WETH9: UniswapContract{
			ABI:      wethArtifact.ABI,
			Bytecode: wethArtifact.Bytecode,
			Name:     "WETH9",
		},
	}

	return contractsData, nil
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

// CalculateInitialTokenPrice calculates the initial token price based on liquidity pool parameters
// Returns the price of 1 token in ETH and the price of 1 ETH in tokens
func CalculateInitialTokenPrice(tokenAmount, ethAmount string, tokenDecimals uint8) (pricePerTokenInETH, pricePerETHInTokens *big.Float, err error) {
	// Parse token amount
	tokenAmountBig, ok := new(big.Int).SetString(tokenAmount, 10)
	if !ok {
		return nil, nil, fmt.Errorf("invalid token amount: %s", tokenAmount)
	}

	// Parse ETH amount (assuming wei units)
	ethAmountBig, ok := new(big.Int).SetString(ethAmount, 10)
	if !ok {
		return nil, nil, fmt.Errorf("invalid ETH amount: %s", ethAmount)
	}

	// Convert to float for price calculation
	tokenAmountFloat := new(big.Float).SetInt(tokenAmountBig)
	ethAmountFloat := new(big.Float).SetInt(ethAmountBig)

	// Account for decimals
	tokenDivisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(tokenDecimals)), nil))
	ethDivisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) // ETH has 18 decimals

	// Convert to actual amounts (not wei/smallest units)
	actualTokenAmount := new(big.Float).Quo(tokenAmountFloat, tokenDivisor)
	actualETHAmount := new(big.Float).Quo(ethAmountFloat, ethDivisor)

	// Calculate price of 1 token in ETH: ETH_amount / token_amount
	pricePerTokenInETH = new(big.Float).Quo(actualETHAmount, actualTokenAmount)

	// Calculate price of 1 ETH in tokens: token_amount / ETH_amount
	pricePerETHInTokens = new(big.Float).Quo(actualTokenAmount, actualETHAmount)

	return pricePerTokenInETH, pricePerETHInTokens, nil
}

// FormatTokenPrice formats a token price for display
func FormatTokenPrice(price *big.Float, decimals int) string {
	if price == nil {
		return "0"
	}
	return price.Text('f', decimals)
}

// CalculatePriceImpact calculates the price impact of a swap
// Returns the price impact as a percentage
func CalculatePriceImpact(inputAmount, outputAmount, inputReserve, outputReserve string) (*big.Float, error) {
	// Parse all amounts
	inputBig, ok := new(big.Int).SetString(inputAmount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid input amount: %s", inputAmount)
	}

	outputBig, ok := new(big.Int).SetString(outputAmount, 10)
	if !ok {
		return nil, fmt.Errorf("invalid output amount: %s", outputAmount)
	}

	inputReserveBig, ok := new(big.Int).SetString(inputReserve, 10)
	if !ok {
		return nil, fmt.Errorf("invalid input reserve: %s", inputReserve)
	}

	outputReserveBig, ok := new(big.Int).SetString(outputReserve, 10)
	if !ok {
		return nil, fmt.Errorf("invalid output reserve: %s", outputReserve)
	}

	// Convert to float for calculation
	inputFloat := new(big.Float).SetInt(inputBig)
	outputFloat := new(big.Float).SetInt(outputBig)
	inputReserveFloat := new(big.Float).SetInt(inputReserveBig)
	outputReserveFloat := new(big.Float).SetInt(outputReserveBig)

	// Calculate spot price before swap: outputReserve / inputReserve
	spotPrice := new(big.Float).Quo(outputReserveFloat, inputReserveFloat)

	// Calculate execution price: outputAmount / inputAmount
	executionPrice := new(big.Float).Quo(outputFloat, inputFloat)

	// Calculate price impact: ((spotPrice - executionPrice) / spotPrice) * 100
	priceDiff := new(big.Float).Sub(spotPrice, executionPrice)
	impact := new(big.Float).Quo(priceDiff, spotPrice)
	impactPercentage := new(big.Float).Mul(impact, big.NewFloat(100))

	return impactPercentage, nil
}

// CalculateMinimumLiquidityAmounts calculates minimum amounts for adding liquidity with slippage
func CalculateMinimumLiquidityAmounts(amount0, amount1 string, slippagePercent float64) (min0, min1 string, err error) {
	// Parse amounts
	amount0Big, ok := new(big.Int).SetString(amount0, 10)
	if !ok {
		return "", "", fmt.Errorf("invalid amount0: %s", amount0)
	}

	amount1Big, ok := new(big.Int).SetString(amount1, 10)
	if !ok {
		return "", "", fmt.Errorf("invalid amount1: %s", amount1)
	}

	// Calculate slippage factor (e.g., 0.5% slippage = 0.995 factor)
	slippageFactor := 1.0 - (slippagePercent / 100.0)

	// Apply slippage
	min0Float := new(big.Float).SetInt(amount0Big)
	min0Float.Mul(min0Float, big.NewFloat(slippageFactor))
	min0Big, _ := min0Float.Int(nil)

	min1Float := new(big.Float).SetInt(amount1Big)
	min1Float.Mul(min1Float, big.NewFloat(slippageFactor))
	min1Big, _ := min1Float.Int(nil)

	return min0Big.String(), min1Big.String(), nil
}
