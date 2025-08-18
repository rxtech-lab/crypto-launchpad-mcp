package api

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rxtech-lab/launchpad-mcp/internal/contracts"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
)

// handleUniswapDeploymentPage serves the Uniswap deployment signing page
func (s *APIServer) handleUniswapDeploymentPage(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")
	if sessionID == "" {
		return c.Status(400).SendString("Session ID is required")
	}

	// Validate session
	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).SendString("Session not found or expired")
	}

	if session.SessionType != "deploy_uniswap" {
		return c.Status(400).SendString("Invalid session type")
	}

	// Parse session data and prepare transaction data for embedding
	var sessionData map[string]interface{}
	var transactionData map[string]interface{}

	if err := json.Unmarshal([]byte(session.TransactionData), &sessionData); err == nil {
		if deploymentIDFloat, ok := sessionData["uniswap_deployment_id"].(float64); ok {
			deploymentID := uint(deploymentIDFloat)

			// Get Uniswap deployment record
			if deployment, err := s.db.GetUniswapDeploymentByID(deploymentID); err == nil {
				// Get contract artifacts for client-side deployment
				contractData := make(map[string]interface{})
				contractNames := []string{"WETH9", "Factory", "Router"}

				for _, contractName := range contractNames {
					if artifact, err := contracts.GetContractArtifact(contractName); err == nil {
						// Ensure bytecode has 0x prefix
						bytecode := artifact.Bytecode
						if !strings.HasPrefix(bytecode, "0x") {
							bytecode = "0x" + bytecode
						}

						contractData[contractName] = map[string]interface{}{
							"bytecode": bytecode,
							"abi":      artifact.ABI,
						}
					}
				}

				// Prepare embedded transaction data
				transactionData = map[string]interface{}{
					"session_id":          sessionID,
					"deployment_id":       deployment.ID,
					"version":             deployment.Version,
					"chain_type":          deployment.ChainType,
					"chain_id":            deployment.ChainID,
					"status":              deployment.Status,
					"session_type":        session.SessionType,
					"metadata":            sessionData["metadata"],
					"deployment_data":     sessionData["deployment_data"],
					"contracts_to_deploy": contractNames,
					"deployment_order":    "1. WETH9 → 2. Factory → 3. Router",
					"contract_data":       contractData,
				}
			}
		}
	}

	// Serve HTML page with embedded transaction data
	html := s.renderTemplate("deploy_uniswap", map[string]interface{}{
		"SessionID":       session.ID,
		"TransactionData": transactionData,
	})
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

// handleUniswapDeploymentAPI provides session data for the Uniswap deployment
func (s *APIServer) handleUniswapDeploymentAPI(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")
	if sessionID == "" {
		return c.Status(400).JSON(map[string]string{
			"error": "Session ID is required",
		})
	}

	// Get session from database
	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(map[string]string{
			"error": "Session not found or expired",
		})
	}

	// Verify session type
	if session.SessionType != "deploy_uniswap" {
		return c.Status(400).JSON(map[string]string{
			"error": "Invalid session type for Uniswap deployment",
		})
	}

	// Parse session data
	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(session.TransactionData), &sessionData); err != nil {
		return c.Status(500).JSON(map[string]string{
			"error": "Failed to parse session data",
		})
	}

	// Get Uniswap deployment record
	deploymentIDFloat, ok := sessionData["uniswap_deployment_id"].(float64)
	if !ok {
		return c.Status(500).JSON(map[string]string{
			"error": "Invalid deployment ID in session",
		})
	}

	deploymentID := uint(deploymentIDFloat)
	deployment, err := s.db.GetUniswapDeploymentByID(deploymentID)
	if err != nil {
		return c.Status(404).JSON(map[string]string{
			"error": "Uniswap deployment not found",
		})
	}

	// Get contract artifacts for client-side deployment
	contractData := make(map[string]interface{})
	contractNames := []string{"WETH9", "Factory", "Router"}

	for _, contractName := range contractNames {
		artifact, err := contracts.GetContractArtifact(contractName)
		if err != nil {
			log.Printf("Warning: Could not get artifact for %s: %v", contractName, err)
			continue
		}

		// Ensure bytecode has 0x prefix
		bytecode := artifact.Bytecode
		if !strings.HasPrefix(bytecode, "0x") {
			bytecode = "0x" + bytecode
		}

		contractData[contractName] = map[string]interface{}{
			"bytecode": bytecode,
			"abi":      artifact.ABI,
		}
	}

	// Prepare response data
	response := map[string]interface{}{
		"session_id":          sessionID,
		"deployment_id":       deployment.ID,
		"version":             deployment.Version,
		"chain_type":          deployment.ChainType,
		"chain_id":            deployment.ChainID,
		"status":              deployment.Status,
		"session_type":        session.SessionType,
		"metadata":            sessionData["metadata"],
		"deployment_data":     sessionData["deployment_data"],
		"contracts_to_deploy": contractNames,
		"deployment_order":    "1. WETH9 → 2. Factory → 3. Router",
		"contract_data":       contractData,
	}

	return c.JSON(response)
}

// handleUniswapDeploymentConfirm handles the confirmation of Uniswap deployment transactions
func (s *APIServer) handleUniswapDeploymentConfirm(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")
	if sessionID == "" {
		return c.Status(400).JSON(map[string]string{
			"error": "Session ID is required",
		})
	}

	// Parse request body
	var req struct {
		TransactionHashes map[string]string `json:"transaction_hashes"`
		ContractAddresses map[string]string `json:"contract_addresses"`
		DeployerAddress   string            `json:"deployer_address"`
		Status            string            `json:"status"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(map[string]string{
			"error": "Invalid request body",
		})
	}

	// Get session from database
	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(map[string]string{
			"error": "Session not found or expired",
		})
	}

	// Parse session data to get deployment ID
	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(session.TransactionData), &sessionData); err != nil {
		return c.Status(500).JSON(map[string]string{
			"error": "Failed to parse session data",
		})
	}

	deploymentIDFloat, ok := sessionData["uniswap_deployment_id"].(float64)
	if !ok {
		return c.Status(500).JSON(map[string]string{
			"error": "Invalid deployment ID in session",
		})
	}

	deploymentID := uint(deploymentIDFloat)

	// Verify transactions on-chain if provided
	if len(req.TransactionHashes) > 0 {
		for contractType, txHash := range req.TransactionHashes {
			if txHash != "" {
				log.Printf("Verifying %s deployment transaction: %s", contractType, txHash)
				if err := s.verifyTransactionOnChain(txHash, session.ChainID); err != nil {
					log.Printf("Transaction verification failed for %s: %v", contractType, err)
					// Continue with other verifications - some may succeed
				}
			}
		}
	}

	// Update Uniswap deployment record
	addresses := req.ContractAddresses
	if req.DeployerAddress != "" {
		if addresses == nil {
			addresses = make(map[string]string)
		}
		addresses["deployer"] = req.DeployerAddress
	}

	err = s.db.UpdateUniswapDeploymentStatus(deploymentID, req.Status, addresses, req.TransactionHashes)
	if err != nil {
		return c.Status(500).JSON(map[string]string{
			"error": "Failed to update deployment status",
		})
	}

	// Update session status
	var sessionTxHash string
	if req.TransactionHashes != nil && len(req.TransactionHashes) > 0 {
		// Use factory transaction as the primary one
		if factoryTx, ok := req.TransactionHashes["factory"]; ok {
			sessionTxHash = factoryTx
		} else {
			// Use any available transaction hash
			for _, txHash := range req.TransactionHashes {
				if txHash != "" {
					sessionTxHash = txHash
					break
				}
			}
		}
	}

	err = s.db.UpdateTransactionSessionStatus(sessionID, req.Status, sessionTxHash)
	if err != nil {
		log.Printf("Failed to update session status: %v", err)
		// Don't return error - deployment was updated successfully
	}

	// If deployment is confirmed, auto-configure Uniswap settings
	if req.Status == "confirmed" && req.ContractAddresses != nil {
		deployment, err := s.db.GetUniswapDeploymentByID(deploymentID)
		if err == nil {
			s.configureUniswapSettings(deployment)
		}
	}

	response := map[string]interface{}{
		"success":       true,
		"session_id":    sessionID,
		"deployment_id": deploymentID,
		"status":        req.Status,
		"message":       "Uniswap deployment status updated successfully",
	}

	return c.JSON(response)
}

// configureUniswapSettings automatically configures Uniswap settings after successful deployment
func (s *APIServer) configureUniswapSettings(deployment *models.UniswapDeployment) {
	if deployment.FactoryAddress == "" || deployment.RouterAddress == "" || deployment.WETHAddress == "" {
		log.Printf("Cannot configure Uniswap settings: missing contract addresses")
		return
	}

	err := s.db.SetUniswapConfiguration(
		deployment.Version,
		deployment.RouterAddress,
		deployment.FactoryAddress,
		deployment.WETHAddress,
		"", // quoter_address (v2 doesn't need this)
		"", // position_manager (v2 doesn't need this)
		"", // swap_router02 (v2 doesn't need this)
	)

	if err != nil {
		log.Printf("Failed to auto-configure Uniswap settings: %v", err)
	} else {
		log.Printf("Successfully auto-configured Uniswap %s settings", deployment.Version)
	}
}
