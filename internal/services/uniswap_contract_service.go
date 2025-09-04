package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rxtech-lab/launchpad-mcp/internal/contracts"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

const EthTokenAddress = "0x0000000000000000000000000000000000000000"
const MetadataToken0Address = "token0_address"
const MetadataToken1Address = "token1_address"

type UniswapContractService interface {
	GetPairAddress(token0Address, token1Address string, chain *models.Chain) (string, error)
}

type uniswapContractService struct {
	uniswapService UniswapService
}

func NewUniswapContractService(uniswapService UniswapService) UniswapContractService {
	return &uniswapContractService{
		uniswapService: uniswapService,
	}
}

// GetPairAddress calls the Uniswap Factory contract to get the pair address for two tokens
func (u *uniswapContractService) GetPairAddress(token0Address, token1Address string, chain *models.Chain) (string, error) {
	// Get Uniswap deployment info
	uniswapDeployment, err := u.uniswapService.GetUniswapDeploymentByChain(chain.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get uniswap deployment: %w", err)
	}

	// Convert ETH to WETH address if needed
	token0 := u.convertETHToWETH(token0Address, uniswapDeployment.WETHAddress)
	token1 := u.convertETHToWETH(token1Address, uniswapDeployment.WETHAddress)

	// Get Factory contract ABI
	factoryArtifact, err := contracts.GetFactoryArtifact()
	if err != nil {
		return "", fmt.Errorf("failed to get factory artifact: %w", err)
	}

	factoryABI, err := json.Marshal(factoryArtifact.ABI)
	if err != nil {
		return "", fmt.Errorf("failed to marshal factory ABI: %w", err)
	}

	// Create RPC client
	rpcClient := utils.NewRPCClient(chain.RPC)

	// Call getPair function
	pairAddress, err := u.callGetPair(rpcClient, uniswapDeployment.FactoryAddress, string(factoryABI), token0, token1)
	if err != nil {
		return "", fmt.Errorf("failed to call getPair: %w", err)
	}

	return pairAddress, nil
}

// convertETHToWETH converts "eth" to WETH address
func (u *uniswapContractService) convertETHToWETH(tokenAddress, wethAddress string) string {
	if strings.ToLower(tokenAddress) == EthTokenAddress {
		return wethAddress
	}
	return tokenAddress
}

// callGetPair calls the getPair function on the Uniswap Factory contract
func (u *uniswapContractService) callGetPair(rpcClient *utils.RPCClient, factoryAddress, factoryABI, token0, token1 string) (string, error) {
	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(factoryABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse factory ABI: %w", err)
	}

	// Encode the function call
	data, err := parsedABI.Pack("getPair", common.HexToAddress(token0), common.HexToAddress(token1))
	if err != nil {
		return "", fmt.Errorf("failed to encode getPair call: %w", err)
	}

	// Make the eth_call
	callParams := map[string]interface{}{
		"to":   factoryAddress,
		"data": "0x" + common.Bytes2Hex(data),
	}

	response, err := rpcClient.Call("eth_call", []interface{}{callParams, "latest"})
	if err != nil {
		return "", fmt.Errorf("failed to make eth_call: %w", err)
	}

	// Parse the response
	resultStr, ok := response.Result.(string)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	if resultStr == "0x" || resultStr == "0x0000000000000000000000000000000000000000000000000000000000000000" {
		return "", fmt.Errorf("pair does not exist")
	}

	// Decode the result (address)
	resultBytes := common.FromHex(resultStr)
	if len(resultBytes) < 32 {
		return "", fmt.Errorf("invalid result length")
	}

	// Extract address from the last 20 bytes
	addressBytes := resultBytes[12:32]
	pairAddress := common.BytesToAddress(addressBytes).Hex()

	return pairAddress, nil
}
