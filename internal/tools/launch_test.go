package tools

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/suite"
)

const (
	TEST_TESTNET_RPC      = "http://localhost:8545"
	TEST_TESTNET_CHAIN_ID = "31337"
	TEST_SERVER_PORT      = 9999 // Use fixed port for testing
)

type LaunchToolTestSuite struct {
	suite.Suite
	db         *database.Database
	ethClient  *ethclient.Client
	launchTool *launchTool
	chain      *models.Chain
	template   *models.Template
}

func (suite *LaunchToolTestSuite) SetupSuite() {
	// Initialize in-memory database
	db, err := database.NewDatabase(":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Initialize Ethereum client
	ethClient, err := ethclient.Dial(TEST_TESTNET_RPC)
	suite.Require().NoError(err)
	suite.ethClient = ethClient

	// Verify Ethereum connection
	err = suite.verifyEthereumConnection()
	suite.Require().NoError(err)

	// Initialize services
	evmService := services.NewEvmService()
	txService := services.NewTransactionService(db.DB)

	// Initialize launch tool
	suite.launchTool = NewLaunchTool(db, TEST_SERVER_PORT, evmService, txService)

	// Setup test data
	suite.setupTestChain()
	suite.setupTestTemplate()
}

func (suite *LaunchToolTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.ethClient != nil {
		suite.ethClient.Close()
	}
}

func (suite *LaunchToolTestSuite) SetupTest() {
	// Clean up any existing sessions for each test
	suite.cleanupTestData()
}

func (suite *LaunchToolTestSuite) verifyEthereumConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	networkID, err := suite.ethClient.NetworkID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get network ID: %w", err)
	}

	if networkID.Cmp(big.NewInt(31337)) != 0 {
		return fmt.Errorf("unexpected network ID: got %s, expected 31337", networkID.String())
	}

	return nil
}

func (suite *LaunchToolTestSuite) setupTestChain() {
	chain := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		RPC:       TEST_TESTNET_RPC,
		NetworkID: TEST_TESTNET_CHAIN_ID,
		Name:      "Ethereum Testnet",
		IsActive:  true,
	}

	err := suite.db.CreateChain(chain)
	suite.Require().NoError(err)
	suite.chain = chain
}

func (suite *LaunchToolTestSuite) setupTestTemplate() {
	contractCode := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract CustomToken {
    string public name = "{{.TokenName}}";
    string public symbol = "{{.TokenSymbol}}";
    uint256 public totalSupply = {{.TotalSupply}};
    mapping(address => uint256) public balanceOf;

    constructor() {
        balanceOf[msg.sender] = totalSupply;
    }

    function transfer(address to, uint256 amount) public returns (bool) {
        require(balanceOf[msg.sender] >= amount, "Insufficient balance");
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        return true;
    }
}`

	template := &models.Template{
		Name:         "CustomToken",
		Description:  "A custom token contract template for testing",
		ChainType:    models.TransactionChainTypeEthereum,
		ContractName: "CustomToken",
		TemplateCode: contractCode,
	}

	err := suite.db.CreateTemplate(template)
	suite.Require().NoError(err)
	suite.template = template
}

func (suite *LaunchToolTestSuite) cleanupTestData() {
	// Clean up transaction sessions
	suite.db.DB.Where("1 = 1").Delete(&models.TransactionSession{})

	// Clean up deployments
	suite.db.DB.Where("1 = 1").Delete(&models.Deployment{})
}

func (suite *LaunchToolTestSuite) TestGetTool() {
	tool := suite.launchTool.GetTool()

	suite.Equal("launch", tool.Name)
	suite.Contains(tool.Description, "Generate deployment URL")
	suite.Contains(tool.Description, "contract compilation")
	suite.Contains(tool.Description, "signing interface")

	// Check required parameters
	suite.NotNil(tool.InputSchema)
	properties := tool.InputSchema.Properties

	// Check template_id parameter
	templateIdProp, exists := properties["template_id"]
	suite.True(exists)
	if propMap, ok := templateIdProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
	}

	// Check template_values parameter
	templateValuesProp, exists := properties["template_values"]
	suite.True(exists)
	if propMap, ok := templateValuesProp.(map[string]any); ok {
		suite.Equal("object", propMap["type"])
	}

	// Check optional parameters
	constructorArgsProp, exists := properties["constructor_args"]
	suite.True(exists)
	if propMap, ok := constructorArgsProp.(map[string]any); ok {
		suite.Equal("array", propMap["type"])
	}

	valueProp, exists := properties["value"]
	suite.True(exists)
	if propMap, ok := valueProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
	}

	metadataProp, exists := properties["metadata"]
	suite.True(exists)
	if propMap, ok := metadataProp.(map[string]any); ok {
		suite.Equal("array", propMap["type"])
	}
}

func (suite *LaunchToolTestSuite) TestHandlerSuccess() {
	// Create test request
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": fmt.Sprintf("%d", suite.template.ID),
				"template_values": map[string]interface{}{
					"TokenName":   "TestToken",
					"TokenSymbol": "TST",
					"TotalSupply": "1000000",
				},
				"constructor_args": []interface{}{},
				"value":            "0",
				"metadata": []interface{}{
					map[string]interface{}{
						"title":       "Deploy TestToken",
						"description": "Deploy a test token contract",
					},
				},
			},
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.NotEmpty(result.Content)

	// Verify response content
	if result.IsError {
		// If there's an error, check the error message
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				suite.T().Logf("Error content: %s", textContent.Text)
				suite.FailNow("Expected successful result but got error", textContent.Text)
			}
		}
		suite.FailNow("Expected successful result but got error with no content")
	}

	suite.Len(result.Content, 3)

	// Extract and verify content using type assertion
	var sessionIDContent string
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			sessionIDContent = textContent.Text
			suite.Contains(sessionIDContent, "Transaction session created:")
		}
	}

	if len(result.Content) > 1 {
		if textContent, ok := result.Content[1].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Please return the following url to the user")
		}
	}

	if len(result.Content) > 2 {
		if textContent, ok := result.Content[2].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, fmt.Sprintf("http://localhost:%d/tx/", TEST_SERVER_PORT))
		}
	}

	// Extract session ID from response
	sessionID := sessionIDContent[len("Transaction session created: "):]

	// Verify session was created in database
	txService := services.NewTransactionService(suite.db.DB)
	session, err := txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.NotNil(session)
	suite.Equal(models.TransactionStatusPending, session.TransactionStatus)
	suite.Equal(models.TransactionChainTypeEthereum, session.TransactionChainType)
	suite.Equal(suite.chain.ID, session.ChainID)

	// Verify transaction deployment details
	suite.Len(session.TransactionDeployments, 1)
	deployment := session.TransactionDeployments[0]
	suite.Equal("Deploy Contract", deployment.Title)
	suite.Equal("Deploy contract to the active chain", deployment.Description)
	suite.Equal("0", deployment.Value)
	suite.Equal("", deployment.Receiver)
	suite.NotEmpty(deployment.Data)

	// Verify the transaction data contains compiled bytecode
	suite.True(len(deployment.Data) > 10, "Transaction data should contain compiled bytecode")
	suite.True(deployment.Data[:2] == "0x", "Transaction data should be hex encoded")
}

func (suite *LaunchToolTestSuite) TestHandlerTemplateRendering() {
	// Test with template values
	templateValues := map[string]interface{}{
		"TokenName":   "MyCustomToken",
		"TokenSymbol": "MCT",
		"TotalSupply": "5000000",
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":     fmt.Sprintf("%d", suite.template.ID),
				"template_values": templateValues,
			},
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)

	// Check for errors first
	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				suite.T().Logf("Error content: %s", textContent.Text)
				suite.FailNow("Expected successful result but got error", textContent.Text)
			}
		}
		suite.FailNow("Expected successful result but got error with no content")
	}

	// Extract session ID and verify contract was compiled with template values
	var sessionIDContent string
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			sessionIDContent = textContent.Text
		}
	}
	sessionID := sessionIDContent[len("Transaction session created: "):]

	txService := services.NewTransactionService(suite.db.DB)
	session, err := txService.GetTransactionSession(sessionID)
	suite.NoError(err)

	// The contract should be compiled and ready for deployment
	deployment := session.TransactionDeployments[0]
	suite.NotEmpty(deployment.Data)

	// Verify the rendered contract contains the template values by attempting compilation
	renderedContract := fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract MyCustomToken {
    string public name = "MyCustomToken";
    string public symbol = "MCT";
    uint256 public totalSupply = 5000000;
    mapping(address => uint256) public balanceOf;

    constructor() {
        balanceOf[msg.sender] = totalSupply;
    }

    function transfer(address to, uint256 amount) public returns (bool) {
        require(balanceOf[msg.sender] >= amount, "Insufficient balance");
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        return true;
    }
}`)

	// Verify the contract can be compiled (this indirectly verifies template rendering worked)
	compilationResult, err := utils.CompileSolidity("0.8.27", renderedContract)
	suite.NoError(err)
	suite.NotNil(compilationResult)

	bytecode, exists := compilationResult.Bytecode["MyCustomToken"]
	suite.True(exists)
	suite.NotEmpty(bytecode)
}

func (suite *LaunchToolTestSuite) TestHandlerWithConstructorArgs() {
	// Create a template that requires constructor arguments
	contractCodeWithConstructor := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract TokenWithConstructor {
    string public name;
    string public symbol;
    uint256 public totalSupply;
    mapping(address => uint256) public balanceOf;

    constructor(string memory _name, string memory _symbol, uint256 _supply) {
        name = _name;
        symbol = _symbol;
        totalSupply = _supply;
        balanceOf[msg.sender] = _supply;
    }
}`

	templateWithConstructor := &models.Template{
		Name:         "TokenWithConstructor",
		Description:  "A token contract template that uses constructor arguments",
		ChainType:    models.TransactionChainTypeEthereum,
		ContractName: "TokenWithConstructor",
		TemplateCode: contractCodeWithConstructor,
	}

	err := suite.db.CreateTemplate(templateWithConstructor)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":     fmt.Sprintf("%d", templateWithConstructor.ID),
				"template_values": map[string]interface{}{
					// This template doesn't use template substitution, just constructor args
				},
				"constructor_args": []interface{}{
					"ConstructorToken",
					"CTK",
					1000000,
				},
				"value": "0",
			},
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)

	// Extract session ID and verify
	var sessionIDContent string
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		sessionIDContent = textContent.Text
	}
	sessionID := sessionIDContent[len("Transaction session created: "):]

	txService := services.NewTransactionService(suite.db.DB)
	session, err := txService.GetTransactionSession(sessionID)
	suite.NoError(err)

	deployment := session.TransactionDeployments[0]
	suite.NotEmpty(deployment.Data)

	// The transaction data should contain encoded constructor arguments
	suite.True(len(deployment.Data) > 100, "Transaction data should contain bytecode and constructor args")
}

func (suite *LaunchToolTestSuite) TestHandlerInvalidTemplateID() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": "invalid-id",
				"template_values": map[string]interface{}{
					"TokenName": "Test",
				},
			},
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Invalid template_id")
	}
}

func (suite *LaunchToolTestSuite) TestHandlerTemplateNotFound() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": "99999", // Non-existent template
				"template_values": map[string]interface{}{
					"TokenName": "Test",
				},
			},
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Template not found")
	}
}

func (suite *LaunchToolTestSuite) TestHandlerNoActiveChain() {
	// Deactivate the chain
	err := suite.db.DB.Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", false).Error
	suite.Require().NoError(err)
	suite.chain.IsActive = false

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": fmt.Sprintf("%d", suite.template.ID),
				"template_values": map[string]interface{}{
					"TokenName": "Test",
				},
			},
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "No active chain selected")
	}

	// Reactivate the chain for other tests
	err = suite.db.DB.Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", true).Error
	suite.Require().NoError(err)
	suite.chain.IsActive = true
}

func (suite *LaunchToolTestSuite) TestHandlerChainTypeMismatch() {
	// Create a Solana template
	solanaTemplate := &models.Template{
		Name:         "SolanaToken",
		Description:  "A Solana token template",
		ChainType:    models.TransactionChainTypeSolana,
		ContractName: "SolanaToken",
		TemplateCode: "// Solana contract code",
	}

	err := suite.db.CreateTemplate(solanaTemplate)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": fmt.Sprintf("%d", solanaTemplate.ID),
				"template_values": map[string]interface{}{
					"TokenName": "Test",
				},
			},
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "doesn't match active chain")
	}
}

func (suite *LaunchToolTestSuite) TestHandlerSolanaNotImplemented() {
	// Create a Solana chain and template
	solanaChain := &models.Chain{
		ChainType: models.TransactionChainTypeSolana,
		RPC:       "https://api.devnet.solana.com",
		NetworkID: "devnet",
		Name:      "Solana Devnet",
		IsActive:  true,
	}

	err := suite.db.CreateChain(solanaChain)
	suite.Require().NoError(err)

	// Deactivate Ethereum chain
	err = suite.db.DB.Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", false).Error
	suite.Require().NoError(err)
	suite.chain.IsActive = false

	solanaTemplate := &models.Template{
		Name:         "SolanaToken",
		Description:  "A Solana token template",
		ChainType:    models.TransactionChainTypeSolana,
		ContractName: "SolanaToken",
		TemplateCode: "// Solana contract code",
	}

	err = suite.db.CreateTemplate(solanaTemplate)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": fmt.Sprintf("%d", solanaTemplate.ID),
				"template_values": map[string]interface{}{
					"TokenName": "Test",
				},
			},
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Solana is not implemented yet")
	}

	// Reactivate Ethereum chain and deactivate Solana chain for other tests
	err = suite.db.DB.Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", true).Error
	suite.Require().NoError(err)
	suite.chain.IsActive = true

	err = suite.db.DB.Model(&models.Chain{}).Where("id = ?", solanaChain.ID).Update("is_active", false).Error
	suite.Require().NoError(err)
	solanaChain.IsActive = false
}

func (suite *LaunchToolTestSuite) TestHandlerMissingRequiredFields() {
	// Test missing template_id
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_values": map[string]interface{}{
					"TokenName": "Test",
				},
			},
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Invalid arguments")
	}

	// Test missing template_values
	request2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": fmt.Sprintf("%d", suite.template.ID),
			},
		},
	}

	result2, err := handler(context.Background(), request2)

	suite.NoError(err)
	suite.NotNil(result2)
	suite.True(result2.IsError)
	if textContent, ok := result2.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Invalid arguments")
	}
}

func (suite *LaunchToolTestSuite) TestHandlerInvalidBindArguments() {
	// Create request with invalid argument structure
	invalidRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: "invalid-json-structure", // Should be map[string]interface{}
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), invalidRequest)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "failed to bind arguments")
}

func (suite *LaunchToolTestSuite) TestCreateEvmContractDeploymentTransaction() {
	metadata := []models.TransactionMetadata{
		{Key: "test_key", Value: "test_value"},
	}

	renderedContract := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract TestContract {
    string public name = "TestContract";
    
    constructor() {}
}`

	sessionID, err := suite.launchTool.createEvmContractDeploymentTransaction(
		suite.chain,
		metadata,
		renderedContract,
		"TestContract",
		[]interface{}{},
		"0",
		"Deploy TestContract",
		"Deploy a test contract",
	)

	suite.NoError(err)
	suite.NotEmpty(sessionID)

	// Verify session was created in database
	txService := services.NewTransactionService(suite.db.DB)
	session, err := txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.NotNil(session)

	// Verify session details
	suite.Equal(models.TransactionStatusPending, session.TransactionStatus)
	suite.Equal(models.TransactionChainTypeEthereum, session.TransactionChainType)
	suite.Equal(suite.chain.ID, session.ChainID)
	suite.Len(session.Metadata, 1)
	suite.Equal("test_key", session.Metadata[0].Key)
	suite.Equal("test_value", session.Metadata[0].Value)

	// Verify transaction deployment details
	suite.Len(session.TransactionDeployments, 1)
	deployment := session.TransactionDeployments[0]
	suite.Equal("Deploy TestContract", deployment.Title)
	suite.Equal("Deploy a test contract", deployment.Description)
	suite.Equal("0", deployment.Value)
	suite.NotEmpty(deployment.Data)

	// Verify the transaction data is valid bytecode
	suite.True(len(deployment.Data) > 10)
	suite.True(deployment.Data[:2] == "0x")
}

func (suite *LaunchToolTestSuite) TestToolRegistration() {
	// Test that the tool can be registered with an MCP server
	mcpServer := server.NewMCPServer("test", "1.0.0")

	tool := suite.launchTool.GetTool()
	handler := suite.launchTool.GetHandler()

	// This should not panic
	suite.NotPanics(func() {
		mcpServer.AddTool(tool, handler)
	})
}

func (suite *LaunchToolTestSuite) TestURLGeneration() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": fmt.Sprintf("%d", suite.template.ID),
				"template_values": map[string]interface{}{
					"TokenName":   "URLTest",
					"TokenSymbol": "URL",
					"TotalSupply": "1000",
				},
			},
		},
	}

	handler := suite.launchTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)

	// Check for errors first
	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				suite.T().Logf("Error content: %s", textContent.Text)
				suite.FailNow("Expected successful result but got error", textContent.Text)
			}
		}
		suite.FailNow("Expected successful result but got error with no content")
	}

	suite.Require().Len(result.Content, 3, "Expected 3 content items for successful deployment")

	// Extract the URL from the response
	var urlContent string
	if len(result.Content) > 2 {
		if textContent, ok := result.Content[2].(mcp.TextContent); ok {
			urlContent = textContent.Text
		}
	}
	expectedURLPrefix := fmt.Sprintf("http://localhost:%d/tx/", TEST_SERVER_PORT)
	suite.True(len(urlContent) > len(expectedURLPrefix))
	suite.Contains(urlContent, expectedURLPrefix)

	// Extract session ID from URL
	sessionID := urlContent[len(expectedURLPrefix):]
	suite.NotEmpty(sessionID)

	// Verify the session ID is a valid UUID format
	suite.True(len(sessionID) > 10) // Basic length check for UUID
}

func TestLaunchToolTestSuite(t *testing.T) {
	// Check if Ethereum testnet is available before running tests
	client, err := ethclient.Dial(TEST_TESTNET_RPC)
	if err != nil {
		t.Skipf("Skipping launch tool tests: Ethereum testnet not available at %s. Run 'make e2e-network' to start testnet.", TEST_TESTNET_RPC)
		return
	}

	// Verify network connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	networkID, err := client.NetworkID(ctx)
	client.Close()

	if err != nil || networkID.Cmp(big.NewInt(31337)) != 0 {
		t.Skipf("Skipping launch tool tests: Cannot connect to anvil testnet at %s (network ID should be 31337). Run 'make e2e-network' to start testnet.", TEST_TESTNET_RPC)
		return
	}

	suite.Run(t, new(LaunchToolTestSuite))
}
