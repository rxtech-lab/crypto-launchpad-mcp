package tests

import (
	"path/filepath"
	"testing"

	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/tools"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

const (
	// Local testnet RPC endpoint for testing
	testRPC     = "http://localhost:8545"
	testChainID = "1337"
)

func TestUniswapUtilities(t *testing.T) {
	t.Run("ValidateUniswapVersion", func(t *testing.T) {
		// Test valid version
		err := utils.ValidateUniswapVersion("v2")
		if err != nil {
			t.Errorf("Expected v2 to be valid, got error: %v", err)
		}

		// Test invalid versions
		err = utils.ValidateUniswapVersion("v3")
		if err == nil {
			t.Error("Expected v3 to be invalid (not yet supported)")
		}

		err = utils.ValidateUniswapVersion("invalid")
		if err == nil {
			t.Error("Expected 'invalid' to be invalid")
		}

		err = utils.ValidateUniswapVersion("")
		if err == nil {
			t.Error("Expected empty string to be invalid")
		}
	})

	t.Run("DeployV2Uniswap", func(t *testing.T) {
		// Test successful deployment data preparation
		deploymentData, err := utils.DeployV2Uniswap("ethereum", testChainID)
		if err != nil {
			t.Fatalf("DeployV2Uniswap failed: %v", err)
		}
		if deploymentData == nil {
			t.Fatal("Deployment data is nil")
		}

		// Check that contracts are included
		if deploymentData.Contracts.Factory.Name == "" {
			t.Error("Factory contract name is empty")
		}
		if deploymentData.Contracts.Router.Name == "" {
			t.Error("Router contract name is empty")
		}
		if deploymentData.Contracts.WETH9.Name == "" {
			t.Error("WETH9 contract name is empty")
		}

		// Check metadata exists
		if len(deploymentData.Metadata) == 0 {
			t.Error("Expected metadata to exist")
		}

		// Verify specific metadata items
		foundDeploymentType := false
		for _, metadata := range deploymentData.Metadata {
			if metadata.Title == "Deployment Type" && metadata.Value == "Uniswap V2 Infrastructure" {
				foundDeploymentType = true
				break
			}
		}
		if !foundDeploymentType {
			t.Error("Expected to find 'Deployment Type' metadata")
		}

		t.Logf("Successfully prepared V2 deployment data with %d metadata items", len(deploymentData.Metadata))

		// Test unsupported chain
		_, err = utils.DeployV2Uniswap("solana", "1")
		if err == nil {
			t.Error("Expected error for solana chain")
		}
		if err != nil && !containsString(err.Error(), "only supported on Ethereum") {
			t.Errorf("Expected Ethereum-only error, got: %v", err)
		}
	})

	t.Run("EstimateGas", func(t *testing.T) {
		gasEstimates := utils.EstimateUniswapV2DeploymentGas()
		if gasEstimates == nil {
			t.Fatal("Gas estimates are nil")
		}

		// Check required fields exist and are reasonable
		requiredFields := map[string]uint64{
			"weth9":   100000,  // At least 100k gas
			"factory": 500000,  // At least 500k gas
			"router":  500000,  // At least 500k gas
			"total":   1000000, // At least 1M gas total
		}

		for field, minGas := range requiredFields {
			if gas, ok := gasEstimates[field]; !ok {
				t.Errorf("Missing gas estimate for %s", field)
			} else if gas < minGas {
				t.Errorf("Gas estimate for %s (%d) is less than minimum expected (%d)", field, gas, minGas)
			}
		}

		// Verify total is sum of components (approximately)
		expectedTotal := gasEstimates["weth9"] + gasEstimates["factory"] + gasEstimates["router"]
		if gasEstimates["total"] != expectedTotal {
			t.Logf("Note: Total gas (%d) differs from sum of components (%d)", gasEstimates["total"], expectedTotal)
		}

		t.Logf("Gas estimates: WETH9=%d, Factory=%d, Router=%d, Total=%d",
			gasEstimates["weth9"], gasEstimates["factory"], gasEstimates["router"], gasEstimates["total"])
	})
}

func TestUniswapDatabaseIntegration(t *testing.T) {
	// Create temporary database for testing
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_uniswap.db")

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Setup test chain configuration
	chain := &models.Chain{
		ChainType: "ethereum",
		RPC:       testRPC,
		ChainID:   testChainID,
		Name:      "Local Testnet",
		IsActive:  true,
	}

	err = db.CreateChain(chain)
	if err != nil {
		t.Fatalf("Failed to create chain: %v", err)
	}

	t.Run("CreateUniswapDeployment", func(t *testing.T) {
		// Test creating a Uniswap deployment record
		deployment := &models.UniswapDeployment{
			Version:   "v2",
			ChainType: "ethereum",
			ChainID:   testChainID,
			Status:    "pending",
		}

		err := db.CreateUniswapDeployment(deployment)
		if err != nil {
			t.Fatalf("Failed to create Uniswap deployment: %v", err)
		}

		if deployment.ID == 0 {
			t.Error("Expected deployment ID to be set after creation")
		}

		t.Logf("Created Uniswap deployment with ID: %d", deployment.ID)
	})

	t.Run("GetUniswapDeploymentByChain", func(t *testing.T) {
		// First create a deployment
		deployment := &models.UniswapDeployment{
			Version:   "v2",
			ChainType: "ethereum",
			ChainID:   testChainID,
			Status:    "confirmed", // Important: must be confirmed to be found
		}

		err := db.CreateUniswapDeployment(deployment)
		if err != nil {
			t.Fatalf("Failed to create deployment: %v", err)
		}

		// Now try to find it
		found, err := db.GetUniswapDeploymentByChain("ethereum", testChainID)
		if err != nil {
			t.Fatalf("Failed to get deployment by chain: %v", err)
		}

		if found.ID != deployment.ID {
			t.Errorf("Expected deployment ID %d, got %d", deployment.ID, found.ID)
		}
		if found.Version != "v2" {
			t.Errorf("Expected version v2, got %s", found.Version)
		}

		t.Logf("Successfully retrieved deployment: ID=%d, Version=%s", found.ID, found.Version)
	})

	t.Run("PreventDuplicateDeployment", func(t *testing.T) {
		// Create a confirmed deployment first
		deployment := &models.UniswapDeployment{
			Version:   "v2",
			ChainType: "ethereum",
			ChainID:   "9999", // Use different chain ID to avoid conflicts
			Status:    "confirmed",
		}

		err := db.CreateUniswapDeployment(deployment)
		if err != nil {
			t.Fatalf("Failed to create first deployment: %v", err)
		}

		// Now check if the deploy tool would detect this
		found, err := db.GetUniswapDeploymentByChain("ethereum", "9999")
		if err != nil {
			t.Fatalf("Should have found existing deployment: %v", err)
		}

		if found == nil {
			t.Error("Expected to find existing deployment")
		} else {
			t.Logf("Correctly found existing deployment: ID=%d", found.ID)
		}
	})
}

func TestDeployUniswapToolCreation(t *testing.T) {
	// Test that we can create the tool without errors
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_tool.db")

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create the tool
	_, handler := tools.NewDeployUniswapTool(db, 8080)
	if handler == nil {
		t.Fatal("Handler is nil")
	}

	// Check that tool was created (can't easily check properties due to interface)
	t.Log("Successfully created deploy_uniswap tool and handler")
}

func TestBalanceQueryBasic(t *testing.T) {
	t.Run("QueryNativeBalance", func(t *testing.T) {
		// Test the balance query function - this will test RPC connectivity if available
		result, err := utils.QueryNativeBalance(testRPC, "0x0000000000000000000000000000000000000000", "ethereum")
		if err != nil {
			// Skip if local testnet is not available
			t.Skipf("Local testnet not available at %s: %v", testRPC, err)
			return
		}

		if result == nil {
			t.Fatal("Balance result is nil")
		}

		if result.Address != "0x0000000000000000000000000000000000000000" {
			t.Errorf("Expected address 0x0000000000000000000000000000000000000000, got %s", result.Address)
		}

		if result.NativeSymbol != "ETH" {
			t.Errorf("Expected native symbol ETH, got %s", result.NativeSymbol)
		}

		if result.ChainType != "ethereum" {
			t.Errorf("Expected chain type ethereum, got %s", result.ChainType)
		}

		t.Logf("Successfully queried balance: %s", result.FormattedBalance)
	})
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
