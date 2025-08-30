package tools

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/contracts"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/suite"
)

const (
	POOL_TEST_TESTNET_RPC      = "http://localhost:8545"
	POOL_TEST_TESTNET_CHAIN_ID = "31337"
	POOL_TEST_SERVER_PORT      = 9999
	// Test private key for Anvil (account #0)
	POOL_TEST_PRIVATE_KEY = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

// DeployedContract represents a deployed contract with its details
type DeployedContract struct {
	Address         common.Address
	TransactionHash common.Hash
	ABI             abi.ABI
	BoundContract   *bind.BoundContract
}

type CreateLiquidityPoolTestSuite struct {
	suite.Suite
	db                *database.Database
	ethClient         *ethclient.Client
	tool              *createLiquidityPoolTool
	chain             *models.Chain
	uniswapSettings   *models.UniswapSettings
	uniswapDeployment *models.UniswapDeployment
	testAccount       *bind.TransactOpts
	testAddress       common.Address

	// Deployed contracts
	testToken       *DeployedContract
	weth9Contract   *DeployedContract
	factoryContract *DeployedContract
	routerContract  *DeployedContract

	// Services
	evmService       services.EvmService
	txService        services.TransactionService
	liquidityService services.LiquidityService
	uniswapService   services.UniswapService
}

func (suite *CreateLiquidityPoolTestSuite) SetupSuite() {
	// Initialize in-memory database
	db, err := database.NewDatabase(":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Initialize Ethereum client
	ethClient, err := ethclient.Dial(POOL_TEST_TESTNET_RPC)
	suite.Require().NoError(err)
	suite.ethClient = ethClient

	// Initialize test account first
	suite.setupTestAccount()

	// Verify Ethereum connection
	err = suite.verifyEthereumConnection()
	suite.Require().NoError(err, "Ethereum testnet should be running on localhost:8545 (run 'make e2e-network')")

	// Initialize services
	suite.evmService = services.NewEvmService()
	suite.txService = services.NewTransactionService(db.DB)
	suite.liquidityService = services.NewLiquidityService(db.DB)
	suite.uniswapService = services.NewUniswapService(db.DB)

	// Initialize tool
	suite.tool = NewCreateLiquidityPoolTool(
		db,
		POOL_TEST_SERVER_PORT,
		suite.evmService,
		suite.txService,
		suite.liquidityService,
		suite.uniswapService,
	)

	// Setup test data
	suite.setupTestChain()
	suite.setupUniswapSettings()

	// Deploy contracts
	suite.deployContracts()

	// Setup Uniswap deployment
	suite.setupUniswapDeployment()
}

func (suite *CreateLiquidityPoolTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.ethClient != nil {
		suite.ethClient.Close()
	}
}

func (suite *CreateLiquidityPoolTestSuite) SetupTest() {
	// Clean up any existing sessions and pools for each test
	suite.cleanupTestData()
}

func (suite *CreateLiquidityPoolTestSuite) verifyEthereumConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	networkID, err := suite.ethClient.NetworkID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get network ID: %w", err)
	}

	if networkID.Cmp(big.NewInt(31337)) != 0 {
		return fmt.Errorf("unexpected network ID: got %s, expected 31337", networkID.String())
	}

	// Check balance
	balance, err := suite.ethClient.BalanceAt(ctx, suite.testAddress, nil)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	if balance.Cmp(big.NewInt(0)) == 0 {
		return fmt.Errorf("test account has zero balance")
	}

	return nil
}

func (suite *CreateLiquidityPoolTestSuite) setupTestAccount() {
	privateKey, err := crypto.HexToECDSA(POOL_TEST_PRIVATE_KEY)
	suite.Require().NoError(err)

	suite.testAddress = crypto.PubkeyToAddress(privateKey.PublicKey)

	chainID := big.NewInt(31337)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	suite.Require().NoError(err)

	auth.GasLimit = 5000000
	auth.GasPrice = big.NewInt(1000000000) // 1 gwei

	suite.testAccount = auth
}

func (suite *CreateLiquidityPoolTestSuite) setupTestChain() {
	chain := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		RPC:       POOL_TEST_TESTNET_RPC,
		NetworkID: POOL_TEST_TESTNET_CHAIN_ID,
		Name:      "Ethereum Testnet",
		IsActive:  true,
	}

	err := suite.db.CreateChain(chain)
	suite.Require().NoError(err)
	suite.chain = chain
}

func (suite *CreateLiquidityPoolTestSuite) setupUniswapSettings() {
	settings := &models.UniswapSettings{
		Version:  "v2",
		IsActive: true,
	}

	err := suite.db.DB.Create(settings).Error
	suite.Require().NoError(err)
	suite.uniswapSettings = settings
}

func (suite *CreateLiquidityPoolTestSuite) deployContracts() {
	suite.T().Log("Deploying contracts...")

	// Deploy OpenZeppelin-based test token
	testToken, err := suite.deployOpenZeppelinToken("TestToken", "TEST", big.NewInt(1000000))
	suite.Require().NoError(err)
	suite.testToken = testToken
	suite.T().Logf("✓ Deployed TestToken at %s", testToken.Address.Hex())

	// Deploy WETH9 from embedded artifact
	weth9, err := suite.deployWETH9()
	suite.Require().NoError(err)
	suite.weth9Contract = weth9
	suite.T().Logf("✓ Deployed WETH9 at %s", weth9.Address.Hex())

	// Deploy UniswapV2Factory from embedded artifact
	factory, err := suite.deployUniswapV2Factory()
	suite.Require().NoError(err)
	suite.factoryContract = factory
	suite.T().Logf("✓ Deployed UniswapV2Factory at %s", factory.Address.Hex())

	// Deploy UniswapV2Router02 from embedded artifact
	router, err := suite.deployUniswapV2Router(factory.Address, weth9.Address)
	suite.Require().NoError(err)
	suite.routerContract = router
	suite.T().Logf("✓ Deployed UniswapV2Router02 at %s", router.Address.Hex())
}

func (suite *CreateLiquidityPoolTestSuite) deployOpenZeppelinToken(name, symbol string, supply *big.Int) (*DeployedContract, error) {
	// OpenZeppelin-based ERC20 token with Ownable
	contractCode := fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin-contracts/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin-contracts/contracts/access/Ownable.sol";

contract TestToken is ERC20, Ownable {
    constructor() ERC20("%s", "%s") Ownable(msg.sender) {
        _mint(msg.sender, %s * 10**decimals());
    }
    
    function mint(address to, uint256 amount) public onlyOwner {
        _mint(to, amount);
    }
}`, name, symbol, supply.String())

	// Compile the contract
	compilationResult, err := utils.CompileSolidity("0.8.27", contractCode)
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	// Get bytecode and ABI
	bytecodeHex, exists := compilationResult.Bytecode["TestToken"]
	if !exists {
		return nil, fmt.Errorf("TestToken bytecode not found in compilation result")
	}

	abiData, exists := compilationResult.Abi["TestToken"]
	if !exists {
		return nil, fmt.Errorf("TestToken ABI not found in compilation result")
	}

	// Deploy the contract
	return suite.deployContract(bytecodeHex, abiData)
}

func (suite *CreateLiquidityPoolTestSuite) deployWETH9() (*DeployedContract, error) {
	// Get WETH9 artifact from embedded contracts
	artifact, err := contracts.GetWETH9Artifact()
	if err != nil {
		return nil, fmt.Errorf("failed to get WETH9 artifact: %w", err)
	}

	// Deploy the contract
	return suite.deployContract(artifact.Bytecode, artifact.ABI)
}

func (suite *CreateLiquidityPoolTestSuite) deployUniswapV2Factory() (*DeployedContract, error) {
	// Get Factory artifact from embedded contracts
	artifact, err := contracts.GetFactoryArtifact()
	if err != nil {
		return nil, fmt.Errorf("failed to get Factory artifact: %w", err)
	}

	// Factory constructor needs feeToSetter address
	// We'll use our test account as the feeToSetter
	abiBytes, err := json.Marshal(artifact.ABI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ABI: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Pack constructor arguments
	constructorData, err := parsedABI.Pack("", suite.testAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to pack constructor arguments: %w", err)
	}

	// Combine bytecode with constructor arguments
	bytecode, err := hex.DecodeString(strings.TrimPrefix(artifact.Bytecode, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode bytecode: %w", err)
	}

	fullBytecode := append(bytecode, constructorData...)
	fullBytecodeHex := "0x" + hex.EncodeToString(fullBytecode)

	// Deploy with constructor arguments already packed
	return suite.deployContractRaw(fullBytecodeHex, artifact.ABI, parsedABI)
}

func (suite *CreateLiquidityPoolTestSuite) deployUniswapV2Router(factoryAddress, wethAddress common.Address) (*DeployedContract, error) {
	// Get Router artifact from embedded contracts
	artifact, err := contracts.GetRouterArtifact()
	if err != nil {
		return nil, fmt.Errorf("failed to get Router artifact: %w", err)
	}

	// Router constructor needs factory and WETH addresses
	abiBytes, err := json.Marshal(artifact.ABI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ABI: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Pack constructor arguments
	constructorData, err := parsedABI.Pack("", factoryAddress, wethAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to pack constructor arguments: %w", err)
	}

	// Combine bytecode with constructor arguments
	bytecode, err := hex.DecodeString(strings.TrimPrefix(artifact.Bytecode, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode bytecode: %w", err)
	}

	fullBytecode := append(bytecode, constructorData...)
	fullBytecodeHex := "0x" + hex.EncodeToString(fullBytecode)

	// Deploy with constructor arguments already packed
	return suite.deployContractRaw(fullBytecodeHex, artifact.ABI, parsedABI)
}

func (suite *CreateLiquidityPoolTestSuite) deployContract(bytecodeHex string, abiData interface{}) (*DeployedContract, error) {
	// Parse ABI
	abiBytes, err := json.Marshal(abiData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ABI: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return suite.deployContractRaw(bytecodeHex, abiData, parsedABI)
}

func (suite *CreateLiquidityPoolTestSuite) deployContractRaw(bytecodeHex string, abiData interface{}, parsedABI abi.ABI) (*DeployedContract, error) {
	// Decode bytecode
	bytecode, err := hex.DecodeString(strings.TrimPrefix(bytecodeHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode bytecode: %w", err)
	}

	// Create deployment transaction
	ctx := context.Background()
	nonce, err := suite.ethClient.PendingNonceAt(ctx, suite.testAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := suite.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	gasLimit := uint64(5000000)
	tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, gasPrice, bytecode)

	// Sign and send transaction
	chainID := big.NewInt(31337)
	privateKey, err := crypto.HexToECDSA(POOL_TEST_PRIVATE_KEY)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = suite.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction receipt
	receipt, err := suite.waitForTransaction(signedTx.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction: %w", err)
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("transaction failed")
	}

	// Create bound contract
	boundContract := bind.NewBoundContract(receipt.ContractAddress, parsedABI, suite.ethClient, suite.ethClient, suite.ethClient)

	return &DeployedContract{
		Address:         receipt.ContractAddress,
		TransactionHash: signedTx.Hash(),
		ABI:             parsedABI,
		BoundContract:   boundContract,
	}, nil
}

func (suite *CreateLiquidityPoolTestSuite) waitForTransaction(txHash common.Hash) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for transaction %s", txHash.Hex())
		default:
			receipt, err := suite.ethClient.TransactionReceipt(ctx, txHash)
			if err == nil {
				return receipt, nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (suite *CreateLiquidityPoolTestSuite) setupUniswapDeployment() {
	deployment := &models.UniswapDeployment{
		ChainID:         suite.chain.ID,
		Version:         "v2",
		Status:          models.TransactionStatusConfirmed,
		WETHAddress:     suite.weth9Contract.Address.Hex(),
		FactoryAddress:  suite.factoryContract.Address.Hex(),
		RouterAddress:   suite.routerContract.Address.Hex(),
		DeployerAddress: suite.testAddress.Hex(),
	}

	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2")
	suite.Require().NoError(err)

	// Update with addresses
	err = suite.uniswapService.UpdateWETHAddress(deploymentID, deployment.WETHAddress)
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateFactoryAddress(deploymentID, deployment.FactoryAddress)
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateRouterAddress(deploymentID, deployment.RouterAddress)
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateDeployerAddress(deploymentID, deployment.DeployerAddress)
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateStatus(deploymentID, models.TransactionStatusConfirmed)
	suite.Require().NoError(err)

	suite.uniswapDeployment, err = suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
}

func (suite *CreateLiquidityPoolTestSuite) cleanupTestData() {
	// Clean up transaction sessions
	suite.db.DB.Where("1 = 1").Delete(&models.TransactionSession{})

	// Clean up liquidity pools
	suite.db.DB.Where("1 = 1").Delete(&models.LiquidityPool{})

	// Clean up liquidity positions
	suite.db.DB.Where("1 = 1").Delete(&models.LiquidityPosition{})
}

// Test cases

func (suite *CreateLiquidityPoolTestSuite) TestGetTool() {
	tool := suite.tool.GetTool()

	suite.Equal("create_liquidity_pool", tool.Name)
	suite.Contains(tool.Description, "Create new Uniswap liquidity pool")
	suite.Contains(tool.Description, "signing interface")

	// Check required parameters
	suite.NotNil(tool.InputSchema)
	properties := tool.InputSchema.Properties

	// Check token_address parameter
	tokenAddressProp, exists := properties["token_address"]
	suite.True(exists)
	if propMap, ok := tokenAddressProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
		suite.Contains(propMap["description"], "Address of the token")
	}

	// Check initial_token_amount parameter
	tokenAmountProp, exists := properties["initial_token_amount"]
	suite.True(exists)
	if propMap, ok := tokenAmountProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
		suite.Contains(propMap["description"], "Initial amount of tokens")
	}

	// Check initial_eth_amount parameter
	ethAmountProp, exists := properties["initial_eth_amount"]
	suite.True(exists)
	if propMap, ok := ethAmountProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
		suite.Contains(propMap["description"], "Initial amount of ETH")
	}

	// Check optional metadata parameter
	metadataProp, exists := properties["metadata"]
	suite.True(exists)
	if propMap, ok := metadataProp.(map[string]any); ok {
		suite.Equal("array", propMap["type"])
	}
}

func (suite *CreateLiquidityPoolTestSuite) TestHandlerSuccess() {
	// Create test request
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":        suite.testToken.Address.Hex(),
				"initial_token_amount": "1000000000000000000", // 1 token
				"initial_eth_amount":   "1000000000000000000", // 1 ETH
				"metadata": []interface{}{
					map[string]interface{}{
						"key":   "Pool Type",
						"value": "Test Pool",
					},
				},
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)

	// Verify response
	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				suite.T().Logf("Unexpected error: %s", textContent.Text)
				suite.FailNow("Expected successful result but got error", textContent.Text)
			}
		}
		suite.FailNow("Expected successful result but got error with no content")
	}

	suite.Len(result.Content, 3)

	// Extract session ID
	var sessionIDContent string
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		sessionIDContent = textContent.Text
		suite.Contains(sessionIDContent, "Transaction session created:")
	}

	if textContent, ok := result.Content[1].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Please sign the liquidity pool creation")
	}

	if textContent, ok := result.Content[2].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, fmt.Sprintf("http://localhost:%d/tx/", POOL_TEST_SERVER_PORT))
	}

	// Extract session ID and verify
	sessionID := strings.TrimPrefix(sessionIDContent, "Transaction session created: ")
	suite.NotEmpty(sessionID)

	// Verify session was created
	session, err := suite.txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.NotNil(session)
	suite.Equal(models.TransactionStatusPending, session.TransactionStatus)

	// Verify pool was created
	pool, err := suite.liquidityService.GetLiquidityPoolByTokenAddress(suite.testToken.Address.Hex())
	suite.NoError(err)
	suite.NotNil(pool)
	suite.Equal(suite.testToken.Address.Hex(), pool.TokenAddress)
	suite.Equal("1000000000000000000", pool.InitialToken0)
	suite.Equal("1000000000000000000", pool.InitialToken1)
	suite.Equal(suite.weth9Contract.Address.Hex(), pool.Token1)
	suite.Equal(models.TransactionStatusPending, pool.Status)
}

func (suite *CreateLiquidityPoolTestSuite) TestPoolCreationWithContractInteraction() {
	// First, let's interact with the deployed token contract
	// Check balance of deployer
	var balance *big.Int
	err := suite.testToken.BoundContract.Call(nil, &[]interface{}{&balance}, "balanceOf", suite.testAddress)
	suite.NoError(err)
	suite.T().Logf("Token balance of deployer: %s", balance.String())
	suite.True(balance.Cmp(big.NewInt(0)) > 0)

	// Check token name and symbol
	var name string
	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&name}, "name")
	suite.NoError(err)
	suite.Equal("TestToken", name)

	var symbol string
	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&symbol}, "symbol")
	suite.NoError(err)
	suite.Equal("TEST", symbol)

	// Approve router to spend tokens (for future liquidity addition)
	approveAmount := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))
	tx, err := suite.testToken.BoundContract.Transact(suite.testAccount, "approve", suite.routerContract.Address, approveAmount)
	suite.NoError(err)
	suite.NotNil(tx)

	// Wait for approval transaction
	receipt, err := suite.waitForTransaction(tx.Hash())
	suite.NoError(err)
	suite.Equal(uint64(1), receipt.Status)

	// Check allowance
	var allowance *big.Int
	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&allowance}, "allowance", suite.testAddress, suite.routerContract.Address)
	suite.NoError(err)
	suite.Equal(approveAmount, allowance)
	suite.T().Logf("Router allowance: %s", allowance.String())

	// Now create the pool
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":        suite.testToken.Address.Hex(),
				"initial_token_amount": "100000000000000000000", // 100 tokens
				"initial_eth_amount":   "5000000000000000000",   // 5 ETH
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)

	// Verify pool was created with correct initial price
	pool, err := suite.liquidityService.GetLiquidityPoolByTokenAddress(suite.testToken.Address.Hex())
	suite.NoError(err)
	suite.NotNil(pool)

	// Calculate expected price (5 ETH / 100 tokens = 0.05 ETH per token)
	pricePerToken, pricePerETH, err := utils.CalculateInitialTokenPrice("100000000000000000000", "5000000000000000000", 18)
	suite.NoError(err)
	suite.NotNil(pricePerToken)
	priceFormatted := utils.FormatTokenPrice(pricePerToken, 2)
	suite.T().Logf("Initial token price: %s ETH per token", priceFormatted)
	suite.Equal("0.05", priceFormatted)
	_ = pricePerETH // Not used but avoid unused variable
}

func (suite *CreateLiquidityPoolTestSuite) TestHandlerNoActiveChain() {
	// Deactivate the chain
	err := suite.db.DB.Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", false).Error
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":        "0x1234567890123456789012345678901234567890",
				"initial_token_amount": "1000000000000000000",
				"initial_eth_amount":   "1000000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "No active chain selected")
	}

	// Reactivate the chain
	err = suite.db.DB.Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", true).Error
	suite.Require().NoError(err)
}

func (suite *CreateLiquidityPoolTestSuite) TestHandlerNoUniswapSettings() {
	// Deactivate Uniswap settings
	err := suite.db.DB.Model(&models.UniswapSettings{}).Where("id = ?", suite.uniswapSettings.ID).Update("is_active", false).Error
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":        suite.testToken.Address.Hex(),
				"initial_token_amount": "1000000000000000000",
				"initial_eth_amount":   "1000000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "No Uniswap version selected")
	}

	// Reactivate settings
	err = suite.db.DB.Model(&models.UniswapSettings{}).Where("id = ?", suite.uniswapSettings.ID).Update("is_active", true).Error
	suite.Require().NoError(err)
}

func (suite *CreateLiquidityPoolTestSuite) TestHandlerMissingWETH() {
	// First, delete the existing deployment
	err := suite.uniswapService.DeleteUniswapDeployment(suite.uniswapDeployment.ID)
	suite.Require().NoError(err)

	// Create a new Uniswap deployment without WETH address
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2")
	suite.Require().NoError(err)

	// Only set factory and router, but not WETH
	err = suite.uniswapService.UpdateFactoryAddress(deploymentID, suite.factoryContract.Address.Hex())
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateRouterAddress(deploymentID, suite.routerContract.Address.Hex())
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":        "0xabcdef1234567890123456789012345678901234",
				"initial_token_amount": "1000000000000000000",
				"initial_eth_amount":   "1000000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "WETH address not found")
	}

	// Clean up the test deployment and restore original
	err = suite.uniswapService.DeleteUniswapDeployment(deploymentID)
	suite.Require().NoError(err)

	// Recreate the original deployment
	suite.setupUniswapDeployment()
}

func (suite *CreateLiquidityPoolTestSuite) TestHandlerExistingPool() {
	// Create a confirmed pool first
	existingPool := &models.LiquidityPool{
		TokenAddress:   suite.testToken.Address.Hex(),
		UniswapVersion: "v2",
		Token0:         suite.testToken.Address.Hex(),
		Token1:         suite.weth9Contract.Address.Hex(),
		InitialToken0:  "1000000000000000000",
		InitialToken1:  "1000000000000000000",
		Status:         models.TransactionStatusConfirmed,
		PairAddress:    "0x1234567890123456789012345678901234567890",
	}

	_, err := suite.liquidityService.CreateLiquidityPool(existingPool)
	suite.Require().NoError(err)

	// Try to create another pool for the same token
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":        suite.testToken.Address.Hex(),
				"initial_token_amount": "2000000000000000000",
				"initial_eth_amount":   "2000000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Liquidity pool already exists")
	}

	// Clean up
	suite.db.DB.Where("token_address = ?", suite.testToken.Address.Hex()).Delete(&models.LiquidityPool{})
}

func (suite *CreateLiquidityPoolTestSuite) TestHandlerInvalidArguments() {
	// Test missing token_address
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"initial_token_amount": "1000000000000000000",
				"initial_eth_amount":   "1000000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Invalid arguments")
	}

	// Test missing initial_token_amount
	request2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":      "0x1234567890123456789012345678901234567890",
				"initial_eth_amount": "1000000000000000000",
			},
		},
	}

	result2, err := handler(context.Background(), request2)
	suite.NoError(err)
	suite.NotNil(result2)
	suite.True(result2.IsError)
}

func (suite *CreateLiquidityPoolTestSuite) TestHandlerNonEthereumChain() {
	// Create a Solana chain and activate it
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

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":        "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263",
				"initial_token_amount": "1000000000",
				"initial_eth_amount":   "1000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Uniswap pools are only supported on Ethereum")
	}

	// Clean up
	err = suite.db.DB.Model(&models.Chain{}).Where("id = ?", solanaChain.ID).Update("is_active", false).Error
	suite.Require().NoError(err)

	err = suite.db.DB.Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", true).Error
	suite.Require().NoError(err)
}

func (suite *CreateLiquidityPoolTestSuite) TestPoolDeletionForPendingPool() {
	// Create a pending pool
	pendingPool := &models.LiquidityPool{
		TokenAddress:   "0xaaaa567890123456789012345678901234567890",
		UniswapVersion: "v2",
		Token0:         "0xaaaa567890123456789012345678901234567890",
		Token1:         suite.weth9Contract.Address.Hex(),
		InitialToken0:  "1000000000000000000",
		InitialToken1:  "1000000000000000000",
		Status:         models.TransactionStatusPending, // Pending status
	}

	poolID, err := suite.liquidityService.CreateLiquidityPool(pendingPool)
	suite.Require().NoError(err)

	// Try to create a new pool for the same token (should delete the pending one)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":        "0xaaaa567890123456789012345678901234567890",
				"initial_token_amount": "2000000000000000000",
				"initial_eth_amount":   "2000000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError) // Should succeed after deleting pending pool

	// Verify old pool was deleted
	_, err = suite.liquidityService.GetLiquidityPool(poolID)
	suite.Error(err) // Should not find the old pool

	// Verify new pool was created
	newPool, err := suite.liquidityService.GetLiquidityPoolByTokenAddress("0xaaaa567890123456789012345678901234567890")
	suite.NoError(err)
	suite.NotNil(newPool)
	suite.Equal("2000000000000000000", newPool.InitialToken0)
	suite.Equal("2000000000000000000", newPool.InitialToken1)

	// Clean up
	suite.db.DB.Where("token_address = ?", "0xaaaa567890123456789012345678901234567890").Delete(&models.LiquidityPool{})
}

func (suite *CreateLiquidityPoolTestSuite) TestDeployedTokenMethods() {
	// Test various ERC20 methods on our deployed token

	// 1. Check total supply
	var totalSupply *big.Int
	err := suite.testToken.BoundContract.Call(nil, &[]interface{}{&totalSupply}, "totalSupply")
	suite.NoError(err)
	expectedSupply := new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18))
	suite.Equal(expectedSupply, totalSupply)
	suite.T().Logf("Total supply: %s", totalSupply.String())

	// 2. Check decimals
	var decimals uint8
	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&decimals}, "decimals")
	suite.NoError(err)
	suite.Equal(uint8(18), decimals)

	// 3. Transfer tokens to another address
	recipient := common.HexToAddress("0x0000000000000000000000000000000000000123")
	transferAmount := big.NewInt(1e18) // 1 token

	tx, err := suite.testToken.BoundContract.Transact(suite.testAccount, "transfer", recipient, transferAmount)
	suite.NoError(err)

	receipt, err := suite.waitForTransaction(tx.Hash())
	suite.NoError(err)
	suite.Equal(uint64(1), receipt.Status)

	// Check recipient balance
	var recipientBalance *big.Int
	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&recipientBalance}, "balanceOf", recipient)
	suite.NoError(err)
	suite.Equal(transferAmount, recipientBalance)
	suite.T().Logf("Recipient balance after transfer: %s", recipientBalance.String())
}

func (suite *CreateLiquidityPoolTestSuite) TestWETH9Interaction() {
	// Test WETH9 deposit and withdrawal

	// 1. Deposit ETH to get WETH
	depositAmount := big.NewInt(1e18) // 1 ETH

	// Create a transaction with value to deposit
	auth := *suite.testAccount
	auth.Value = depositAmount

	tx, err := suite.weth9Contract.BoundContract.Transact(&auth, "deposit")
	suite.NoError(err)

	receipt, err := suite.waitForTransaction(tx.Hash())
	suite.NoError(err)
	suite.Equal(uint64(1), receipt.Status)

	// Check WETH balance
	var wethBalance *big.Int
	err = suite.weth9Contract.BoundContract.Call(nil, &[]interface{}{&wethBalance}, "balanceOf", suite.testAddress)
	suite.NoError(err)
	suite.True(wethBalance.Cmp(depositAmount) >= 0) // Should have at least the deposited amount
	suite.T().Logf("WETH balance after deposit: %s", wethBalance.String())

	// 2. Withdraw WETH to get ETH back
	withdrawAmount := big.NewInt(5e17) // 0.5 ETH

	// Reset auth value for withdrawal
	auth.Value = big.NewInt(0)

	tx, err = suite.weth9Contract.BoundContract.Transact(&auth, "withdraw", withdrawAmount)
	suite.NoError(err)

	receipt, err = suite.waitForTransaction(tx.Hash())
	suite.NoError(err)
	suite.Equal(uint64(1), receipt.Status)

	// Check updated WETH balance
	var newWethBalance *big.Int
	err = suite.weth9Contract.BoundContract.Call(nil, &[]interface{}{&newWethBalance}, "balanceOf", suite.testAddress)
	suite.NoError(err)

	expectedBalance := new(big.Int).Sub(wethBalance, withdrawAmount)
	suite.Equal(expectedBalance, newWethBalance)
	suite.T().Logf("WETH balance after withdrawal: %s", newWethBalance.String())
}

func (suite *CreateLiquidityPoolTestSuite) TestUniswapFactoryInteraction() {
	// Test UniswapV2Factory methods

	// Check if a pair exists (should not exist yet)
	var pairAddress common.Address
	err := suite.factoryContract.BoundContract.Call(nil, &[]interface{}{&pairAddress}, "getPair", suite.testToken.Address, suite.weth9Contract.Address)
	suite.NoError(err)
	suite.Equal(common.Address{}, pairAddress) // Should be zero address since pair doesn't exist
	suite.T().Log("Pair doesn't exist yet (as expected)")

	// Check fee recipient
	var feeToAddress common.Address
	err = suite.factoryContract.BoundContract.Call(nil, &[]interface{}{&feeToAddress}, "feeTo")
	suite.NoError(err)
	suite.T().Logf("Fee recipient: %s", feeToAddress.Hex())

	// Check fee setter (should be our test address)
	var feeSetterAddress common.Address
	err = suite.factoryContract.BoundContract.Call(nil, &[]interface{}{&feeSetterAddress}, "feeToSetter")
	suite.NoError(err)
	suite.Equal(suite.testAddress, feeSetterAddress)
	suite.T().Logf("Fee setter: %s", feeSetterAddress.Hex())
}

func (suite *CreateLiquidityPoolTestSuite) TestToolRegistration() {
	// Test that the tool can be registered with an MCP server
	mcpServer := server.NewMCPServer("test", "1.0.0")

	tool := suite.tool.GetTool()
	handler := suite.tool.GetHandler()

	// This should not panic
	suite.NotPanics(func() {
		mcpServer.AddTool(tool, handler)
	})
}

func (suite *CreateLiquidityPoolTestSuite) TestURLGeneration() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":        suite.testToken.Address.Hex(),
				"initial_token_amount": "5000000000000000000",  // 5 tokens
				"initial_eth_amount":   "10000000000000000000", // 10 ETH
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)
	suite.Require().Len(result.Content, 3)

	// Extract the URL from the response
	var urlContent string
	if textContent, ok := result.Content[2].(mcp.TextContent); ok {
		urlContent = textContent.Text
	}

	expectedURLPrefix := fmt.Sprintf("http://localhost:%d/tx/", POOL_TEST_SERVER_PORT)
	suite.Contains(urlContent, expectedURLPrefix)

	// Extract session ID from URL
	sessionID := strings.TrimPrefix(urlContent, expectedURLPrefix)
	suite.NotEmpty(sessionID)
	suite.True(len(sessionID) > 10) // Basic check for UUID
}

func (suite *CreateLiquidityPoolTestSuite) TestHandlerInvalidBindArguments() {
	// Create request with invalid argument structure
	invalidRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: "invalid-json-structure", // Should be map[string]interface{}
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), invalidRequest)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "failed to bind arguments")
}

func (suite *CreateLiquidityPoolTestSuite) TestPoolPriceCalculation() {
	// Test price calculation with specific amounts
	tokenAmount := "50000000000000000000" // 50 tokens
	ethAmount := "2500000000000000000"    // 2.5 ETH

	// Expected price: 2.5 ETH / 50 tokens = 0.05 ETH per token
	pricePerToken, _, err := utils.CalculateInitialTokenPrice(tokenAmount, ethAmount, 18)
	suite.NoError(err)
	suite.NotNil(pricePerToken)
	priceFormatted := utils.FormatTokenPrice(pricePerToken, 2)
	suite.Equal("0.05", priceFormatted)

	// Create pool with these amounts
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":        suite.testToken.Address.Hex(),
				"initial_token_amount": tokenAmount,
				"initial_eth_amount":   ethAmount,
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)

	// Verify pool was created with correct amounts
	pool, err := suite.liquidityService.GetLiquidityPoolByTokenAddress(suite.testToken.Address.Hex())
	suite.NoError(err)
	suite.Equal(tokenAmount, pool.InitialToken0)
	suite.Equal(ethAmount, pool.InitialToken1)
}

// Test runner
func TestCreateLiquidityPoolTestSuite(t *testing.T) {
	// Check if Ethereum testnet is available before running tests
	client, err := ethclient.Dial(POOL_TEST_TESTNET_RPC)
	if err != nil {
		t.Skipf("Skipping create liquidity pool tests: Ethereum testnet not available at %s. Run 'make e2e-network' to start testnet.", POOL_TEST_TESTNET_RPC)
		return
	}

	// Verify network connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	networkID, err := client.NetworkID(ctx)
	client.Close()

	if err != nil || networkID.Cmp(big.NewInt(31337)) != 0 {
		t.Skipf("Skipping create liquidity pool tests: Cannot connect to anvil testnet at %s (network ID should be 31337). Run 'make e2e-network' to start testnet.", POOL_TEST_TESTNET_RPC)
		return
	}

	suite.Run(t, new(CreateLiquidityPoolTestSuite))
}
