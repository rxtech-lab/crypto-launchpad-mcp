package api

import (
	"encoding/json"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
)

// generateCreatePoolTransactionData generates transaction data for pool creation
func (s *APIServer) generateCreatePoolTransactionData(pool *models.LiquidityPool, sessionData map[string]interface{}, activeChain *models.Chain) map[string]interface{} {
	// Get Uniswap settings
	uniswapSettings, _ := s.db.GetActiveUniswapSettings()

	transactionData := map[string]interface{}{
		"pool_id":              pool.ID,
		"token_address":        pool.TokenAddress,
		"initial_token_amount": pool.InitialToken0,
		"initial_eth_amount":   pool.InitialToken1,
		"uniswap_version":      uniswapSettings.Version,
		"chain_type":           activeChain.ChainType,
		"chain_id":             activeChain.ChainID,
		"rpc":                  activeChain.RPC,
		"session_type":         "create_pool",
	}

	// Add any additional session data that was stored
	for key, value := range sessionData {
		if key != "pool_id" { // Don't overwrite pool_id
			transactionData[key] = value
		}
	}

	return transactionData
}

// generateAddLiquidityTransactionData generates transaction data for adding liquidity
func (s *APIServer) generateAddLiquidityTransactionData(position *models.LiquidityPosition, pool *models.LiquidityPool, sessionData map[string]interface{}, activeChain *models.Chain) map[string]interface{} {
	// Get Uniswap settings
	uniswapSettings, _ := s.db.GetActiveUniswapSettings()

	transactionData := map[string]interface{}{
		"position_id":     position.ID,
		"pool_id":         pool.ID,
		"token_address":   pool.TokenAddress,
		"token_amount":    position.Token0Amount,
		"eth_amount":      position.Token1Amount,
		"uniswap_version": uniswapSettings.Version,
		"chain_type":      activeChain.ChainType,
		"chain_id":        activeChain.ChainID,
		"rpc":             activeChain.RPC,
		"session_type":    "add_liquidity",
	}

	// Add min amounts if present in session data
	if minTokenAmount, ok := sessionData["min_token_amount"]; ok {
		transactionData["min_token_amount"] = minTokenAmount
	}
	if minETHAmount, ok := sessionData["min_eth_amount"]; ok {
		transactionData["min_eth_amount"] = minETHAmount
	}

	return transactionData
}

// generateRemoveLiquidityTransactionData generates transaction data for removing liquidity
func (s *APIServer) generateRemoveLiquidityTransactionData(position *models.LiquidityPosition, pool *models.LiquidityPool, sessionData map[string]interface{}, activeChain *models.Chain) map[string]interface{} {
	// Get Uniswap settings
	uniswapSettings, _ := s.db.GetActiveUniswapSettings()

	transactionData := map[string]interface{}{
		"position_id":      position.ID,
		"pool_id":          pool.ID,
		"token_address":    pool.TokenAddress,
		"liquidity_amount": sessionData["liquidity_amount"],
		"uniswap_version":  uniswapSettings.Version,
		"chain_type":       activeChain.ChainType,
		"chain_id":         activeChain.ChainID,
		"rpc":              activeChain.RPC,
		"session_type":     "remove_liquidity",
	}

	// Add min amounts if present
	if minTokenAmount, ok := sessionData["min_token_amount"]; ok {
		transactionData["min_token_amount"] = minTokenAmount
	}
	if minETHAmount, ok := sessionData["min_eth_amount"]; ok {
		transactionData["min_eth_amount"] = minETHAmount
	}

	return transactionData
}

// generateSwapTransactionData generates transaction data for token swaps
func (s *APIServer) generateSwapTransactionData(swap *models.SwapTransaction, sessionData map[string]interface{}, activeChain *models.Chain) map[string]interface{} {
	// Get Uniswap settings
	uniswapSettings, _ := s.db.GetActiveUniswapSettings()

	transactionData := map[string]interface{}{
		"swap_id":         swap.ID,
		"token_in":        swap.FromToken,
		"token_out":       swap.ToToken,
		"amount_in":       swap.FromAmount,
		"amount_out_min":  sessionData["amount_out_min"],
		"uniswap_version": uniswapSettings.Version,
		"chain_type":      activeChain.ChainType,
		"chain_id":        activeChain.ChainID,
		"rpc":             activeChain.RPC,
		"session_type":    "swap",
	}

	// Add any additional swap parameters
	if deadline, ok := sessionData["deadline"]; ok {
		transactionData["deadline"] = deadline
	}
	if path, ok := sessionData["path"]; ok {
		transactionData["path"] = path
	}

	return transactionData
}

// parseLiquiditySessionData parses session transaction data and returns the relevant IDs
func (s *APIServer) parseLiquiditySessionData(sessionDataJSON string) (map[string]interface{}, error) {
	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(sessionDataJSON), &sessionData); err != nil {
		return nil, err
	}
	return sessionData, nil
}
