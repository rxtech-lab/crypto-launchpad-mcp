package hooks

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rxtech-lab/launchpad-mcp/internal/contracts"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"gorm.io/gorm"
)

type LiquidityPoolHook struct {
	db               *gorm.DB
	liquidityService services.LiquidityService
	uniswapService   services.UniswapService
	chainService     services.ChainService
	txService        services.TransactionService
}

// CanHandle implements Hook.
func (l *LiquidityPoolHook) CanHandle(txType models.TransactionType) bool {
	return txType == models.TransactionTypeLiquidityPoolCreation
}

// OnTransactionConfirmed implements Hook.
func (l *LiquidityPoolHook) OnTransactionConfirmed(txType models.TransactionType, txHash string, contractAddress *string, session models.TransactionSession) error {
	switch txType {
	case models.TransactionTypeLiquidityPoolCreation:
		// only handle the creation of the liquidity pool
		// position should be fetched from the blockchain
		return l.handleLiquidityPoolCreation(txHash, session)
	default:
		return nil
	}
}

// handleLiquidityPoolCreation updates the liquidity pool with the confirmed transaction details
func (l *LiquidityPoolHook) handleLiquidityPoolCreation(txHash string, session models.TransactionSession) error {
	// find the pool by session id
	pool, err := l.liquidityService.GetLiquidityPoolBySessionId(session.ID)
	if err != nil {
		return err
	}

	// Get token addresses from session metadata
	token0Address, token1Address, err := l.getTokenAddressesFromSession(session)
	if err != nil {
		return fmt.Errorf("failed to get token addresses: %w", err)
	}

	// Get pair address from Uniswap Factory contract
	pairAddress, err := l.getPairAddressFromContract(token0Address, token1Address, session)
	if err != nil {
		return fmt.Errorf("failed to get pair address: %w", err)
	}

	// Update the pool with transaction hash and pair address
	return l.liquidityService.UpdateLiquidityPoolStatus(pool.ID, models.TransactionStatusConfirmed, pairAddress, txHash)
}

// getTokenAddressesFromSession extracts token addresses from transaction session metadata
func (l *LiquidityPoolHook) getTokenAddressesFromSession(session models.TransactionSession) (string, string, error) {
	var token0Address, token1Address string

	// Extract from metadata
	for _, meta := range session.Metadata {
		if meta.Key == "token0_address" {
			token0Address = meta.Value
		}
		if meta.Key == "token1_address" {
			token1Address = meta.Value
		}
	}

	if token0Address == "" || token1Address == "" {
		return "", "", fmt.Errorf("token addresses not found in session metadata")
	}

	return token0Address, token1Address, nil
}

// getPairAddressFromContract calls the Uniswap Factory contract to get the pair address
func (l *LiquidityPoolHook) getPairAddressFromContract(token0Address, token1Address string, session models.TransactionSession) (string, error) {
	// Get the active chain
	activeChain, err := l.chainService.GetActiveChain()
	if err != nil {
		return "", fmt.Errorf("failed to get active chain: %w", err)
	}

	// Get Uniswap deployment info
	uniswapDeployment, err := l.uniswapService.GetUniswapDeploymentByChain(activeChain.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get uniswap deployment: %w", err)
	}

	// Convert ETH to WETH address if needed
	token0 := l.convertETHToWETH(token0Address, uniswapDeployment.WETHAddress)
	token1 := l.convertETHToWETH(token1Address, uniswapDeployment.WETHAddress)

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
	rpcClient := utils.NewRPCClient(activeChain.RPC)

	// Call getPair function
	pairAddress, err := l.callGetPair(rpcClient, uniswapDeployment.FactoryAddress, string(factoryABI), token0, token1)
	if err != nil {
		return "", fmt.Errorf("failed to call getPair: %w", err)
	}

	return pairAddress, nil
}

// convertETHToWETH converts "eth" to WETH address
func (l *LiquidityPoolHook) convertETHToWETH(tokenAddress, wethAddress string) string {
	if strings.ToLower(tokenAddress) == "eth" {
		return wethAddress
	}
	return tokenAddress
}

// callGetPair calls the getPair function on the Uniswap Factory contract
func (l *LiquidityPoolHook) callGetPair(rpcClient *utils.RPCClient, factoryAddress, factoryABI, token0, token1 string) (string, error) {
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

func NewLiquidityPoolHook(db *gorm.DB, liquidityService services.LiquidityService, uniswapService services.UniswapService, chainService services.ChainService, txService services.TransactionService) services.Hook {
	return &LiquidityPoolHook{
		db:               db,
		liquidityService: liquidityService,
		uniswapService:   uniswapService,
		chainService:     chainService,
		txService:        txService,
	}
}
