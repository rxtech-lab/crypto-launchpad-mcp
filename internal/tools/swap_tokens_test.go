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
	"github.com/rxtech-lab/launchpad-mcp/internal/contracts"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/suite"
)

const (
	SWAP_TEST_TESTNET_RPC      = "http://localhost:8545"
	SWAP_TEST_TESTNET_CHAIN_ID = "31337"
	SWAP_TEST_SERVER_PORT      = 9998
	// Test private key for Anvil (account #0)
	SWAP_TEST_PRIVATE_KEY = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

type SwapTokensTestSuite struct {
	suite.Suite
	db                services.DBService
	ethClient         *ethclient.Client
	tool              *swapTokensTool
	chain             *models.Chain
	uniswapDeployment *models.UniswapDeployment
	testAccount       *bind.TransactOpts
	testAddress       common.Address

	// Deployed contracts
	testToken       *DeployedContract
	testToken2      *DeployedContract
	weth9Contract   *DeployedContract
	factoryContract *DeployedContract
	routerContract  *DeployedContract

	// Services
	evmService       services.EvmService
	txService        services.TransactionService
	liquidityService services.LiquidityService
	uniswapService   services.UniswapService
	chainService     services.ChainService

	// Liquidity pools for testing
	ethTokenPool     *models.LiquidityPool
	tokenEthPool     *models.LiquidityPool
	token1Token2Pool *models.LiquidityPool
}

func (suite *SwapTokensTestSuite) SetupSuite() {
	// Initialize in-memory database
	db, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Initialize Ethereum client
	ethClient, err := ethclient.Dial(SWAP_TEST_TESTNET_RPC)
	suite.Require().NoError(err)
	suite.ethClient = ethClient

	// Initialize test account first
	suite.setupTestAccount()

	// Verify Ethereum connection
	err = suite.verifyEthereumConnection()
	suite.Require().NoError(err, "Ethereum testnet should be running on localhost:8545 (run 'make e2e-network')")

	// Initialize services
	suite.evmService = services.NewEvmService()
	suite.txService = services.NewTransactionService(db.GetDB())
	suite.liquidityService = services.NewLiquidityService(db.GetDB())
	suite.uniswapService = services.NewUniswapService(db.GetDB())
	suite.chainService = services.NewChainService(db.GetDB())

	// Initialize tool
	suite.tool = NewSwapTokensTool(
		suite.chainService,
		suite.liquidityService,
		suite.uniswapService,
		suite.txService,
		SWAP_TEST_SERVER_PORT,
		suite.evmService,
	)

	// Setup test data
	suite.setupTestChain()

	// Deploy contracts
	suite.deployContracts()

	// Setup Uniswap deployment
	suite.setupUniswapDeployment()

	// Create liquidity pools for testing swaps
	suite.createLiquidityPools()
}

func (suite *SwapTokensTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.ethClient != nil {
		suite.ethClient.Close()
	}
}

func (suite *SwapTokensTestSuite) SetupTest() {
	// Clean up any existing sessions for each test
	suite.cleanupTestData()
}

func (suite *SwapTokensTestSuite) verifyEthereumConnection() error {
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

func (suite *SwapTokensTestSuite) setupTestAccount() {
	privateKey, err := crypto.HexToECDSA(SWAP_TEST_PRIVATE_KEY)
	suite.Require().NoError(err)

	suite.testAddress = crypto.PubkeyToAddress(privateKey.PublicKey)

	chainID := big.NewInt(31337)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	suite.Require().NoError(err)

	auth.GasLimit = 5000000
	auth.GasPrice = big.NewInt(1000000000) // 1 gwei

	suite.testAccount = auth
}

func (suite *SwapTokensTestSuite) setupTestChain() {
	chain := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		RPC:       SWAP_TEST_TESTNET_RPC,
		NetworkID: SWAP_TEST_TESTNET_CHAIN_ID,
		Name:      "Ethereum Testnet",
		IsActive:  true,
	}

	err := suite.chainService.CreateChain(chain)
	suite.Require().NoError(err)
	suite.chain = chain
}

func (suite *SwapTokensTestSuite) deployContracts() {
	suite.T().Log("Deploying contracts...")

	// Deploy OpenZeppelin-based test tokens with larger initial supply for testing
	testToken, err := suite.deployOpenZeppelinToken("TestToken", "TEST", big.NewInt(10000000)) // 10 million tokens
	suite.Require().NoError(err)
	suite.testToken = testToken
	suite.T().Logf("✓ Deployed TestToken at %s", testToken.Address.Hex())

	testToken2, err := suite.deployOpenZeppelinToken("TestToken2", "TEST2", big.NewInt(10000000)) // 10 million tokens
	suite.Require().NoError(err)
	suite.testToken2 = testToken2
	suite.T().Logf("✓ Deployed TestToken2 at %s", testToken2.Address.Hex())

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

func (suite *SwapTokensTestSuite) deployOpenZeppelinToken(name, symbol string, supply *big.Int) (*DeployedContract, error) {
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

func (suite *SwapTokensTestSuite) deployWETH9() (*DeployedContract, error) {
	// Get WETH9 artifact from embedded contracts
	artifact, err := contracts.GetWETH9Artifact()
	if err != nil {
		return nil, fmt.Errorf("failed to get WETH9 artifact: %w", err)
	}

	// Deploy the contract
	return suite.deployContract(artifact.Bytecode, artifact.ABI)
}

func (suite *SwapTokensTestSuite) deployUniswapV2Factory() (*DeployedContract, error) {
	// Get Factory artifact from embedded contracts
	artifact, err := contracts.GetFactoryArtifact()
	if err != nil {
		return nil, fmt.Errorf("failed to get Factory artifact: %w", err)
	}

	// Factory constructor needs feeToSetter address
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

func (suite *SwapTokensTestSuite) deployUniswapV2Router(factoryAddress, wethAddress common.Address) (*DeployedContract, error) {
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

func (suite *SwapTokensTestSuite) deployContract(bytecodeHex string, abiData interface{}) (*DeployedContract, error) {
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

func (suite *SwapTokensTestSuite) deployContractRaw(bytecodeHex string, abiData interface{}, parsedABI abi.ABI) (*DeployedContract, error) {
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
	privateKey, err := crypto.HexToECDSA(SWAP_TEST_PRIVATE_KEY)
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

func (suite *SwapTokensTestSuite) waitForTransaction(txHash common.Hash) (*types.Receipt, error) {
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

func (suite *SwapTokensTestSuite) setupUniswapDeployment() {
	deployment := &models.UniswapDeployment{
		ChainID:         suite.chain.ID,
		Version:         "v2",
		Status:          models.TransactionStatusConfirmed,
		WETHAddress:     suite.weth9Contract.Address.Hex(),
		FactoryAddress:  suite.factoryContract.Address.Hex(),
		RouterAddress:   suite.routerContract.Address.Hex(),
		DeployerAddress: suite.testAddress.Hex(),
	}

	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2", nil)
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

func (suite *SwapTokensTestSuite) createLiquidityPools() {
	suite.T().Log("Creating liquidity pools for testing...")

	// Create ETH-Token1 pool
	suite.createETHTokenPool()

	// Create Token2-ETH pool
	suite.createTokenETHPool()

	// Create Token1-Token2 pool (via WETH)
	suite.createTokenTokenPool()
}

func (suite *SwapTokensTestSuite) createETHTokenPool() {
	// Approve token for router
	err := suite.approveToken(suite.testToken, suite.routerContract.Address, big.NewInt(0).Mul(big.NewInt(1000000), big.NewInt(1e18)))
	suite.Require().NoError(err)

	// Add liquidity ETH-Token1
	tokenAmount := big.NewInt(0).Mul(big.NewInt(100000), big.NewInt(1e18)) // 100K tokens
	ethAmount := big.NewInt(0).Mul(big.NewInt(100), big.NewInt(1e18))      // 100 ETH

	err = suite.addLiquidityETH(suite.testToken.Address, tokenAmount, ethAmount)
	suite.Require().NoError(err)

	suite.T().Logf("✓ Created ETH-Token1 liquidity pool")
}

func (suite *SwapTokensTestSuite) createTokenETHPool() {
	// Approve token2 for router
	err := suite.approveToken(suite.testToken2, suite.routerContract.Address, big.NewInt(0).Mul(big.NewInt(2000000), big.NewInt(1e18)))
	suite.Require().NoError(err)

	// Add liquidity Token2-ETH
	tokenAmount := big.NewInt(0).Mul(big.NewInt(200000), big.NewInt(1e18)) // 200K tokens
	ethAmount := big.NewInt(0).Mul(big.NewInt(50), big.NewInt(1e18))       // 50 ETH

	err = suite.addLiquidityETH(suite.testToken2.Address, tokenAmount, ethAmount)
	suite.Require().NoError(err)

	suite.T().Logf("✓ Created Token2-ETH liquidity pool")
}

func (suite *SwapTokensTestSuite) createTokenTokenPool() {
	// For token-to-token swaps, we'll use the existing ETH pools as intermediary
	// No need to create a direct token1-token2 pool as swaps go through WETH
	suite.T().Logf("✓ Token1-Token2 swaps will use ETH pools as intermediary")
}

func (suite *SwapTokensTestSuite) approveToken(token *DeployedContract, spender common.Address, amount *big.Int) error {
	// Approve token spending
	tx, err := token.BoundContract.Transact(suite.testAccount, "approve", spender, amount)
	if err != nil {
		return fmt.Errorf("failed to approve token: %w", err)
	}

	receipt, err := suite.waitForTransaction(tx.Hash())
	if err != nil {
		return fmt.Errorf("failed to wait for approval: %w", err)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("approval transaction failed")
	}

	return nil
}

func (suite *SwapTokensTestSuite) addLiquidityETH(tokenAddress common.Address, tokenAmount, ethAmount *big.Int) error {
	// Add liquidity with ETH
	deadline := big.NewInt(time.Now().Unix() + 600)

	// Set transaction value (ETH amount)
	opts := *suite.testAccount
	opts.Value = ethAmount

	tx, err := suite.routerContract.BoundContract.Transact(&opts, "addLiquidityETH",
		tokenAddress,
		tokenAmount,
		big.NewInt(0), // amountTokenMin (0 for test)
		big.NewInt(0), // amountETHMin (0 for test)
		suite.testAddress,
		deadline,
	)
	if err != nil {
		return fmt.Errorf("failed to add liquidity: %w", err)
	}

	receipt, err := suite.waitForTransaction(tx.Hash())
	if err != nil {
		return fmt.Errorf("failed to wait for liquidity: %w", err)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("add liquidity transaction failed")
	}

	return nil
}

func (suite *SwapTokensTestSuite) cleanupTestData() {
	// Clean up transaction sessions
	suite.db.GetDB().Where("1 = 1").Delete(&models.TransactionSession{})

	// Clean up swap transactions
	suite.db.GetDB().Where("1 = 1").Delete(&models.SwapTransaction{})
}

func (suite *SwapTokensTestSuite) executeTransaction(data, value, to string) (*types.Receipt, error) {
	ctx := context.Background()

	// Get nonce
	nonce, err := suite.ethClient.PendingNonceAt(ctx, suite.testAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Parse value
	txValue := big.NewInt(0)
	if value != "" && value != "0" {
		var ok bool
		txValue, ok = new(big.Int).SetString(value, 10)
		if !ok {
			return nil, fmt.Errorf("invalid value: %s", value)
		}
	}

	// Decode data
	txData, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	// Get gas price
	gasPrice, err := suite.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create transaction
	var tx *types.Transaction
	if to == "" {
		// Contract deployment
		tx = types.NewContractCreation(nonce, txValue, uint64(3000000), gasPrice, txData)
	} else {
		// Contract call
		toAddr := common.HexToAddress(to)
		tx = types.NewTransaction(nonce, toAddr, txValue, uint64(3000000), gasPrice, txData)
	}

	// Sign transaction
	chainID := big.NewInt(31337)
	privateKey, err := crypto.HexToECDSA(SWAP_TEST_PRIVATE_KEY)
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = suite.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for receipt
	receipt, err := suite.waitForTransaction(signedTx.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction: %w", err)
	}

	return receipt, nil
}

// Test cases

func (suite *SwapTokensTestSuite) TestGetTool() {
	tool := suite.tool.GetTool()

	suite.Equal("swap_tokens", tool.Name)
	suite.Contains(tool.Description, "Execute token swaps via Uniswap")
	suite.Contains(tool.Description, "signing interface")

	// Check required parameters exist
	suite.NotNil(tool.InputSchema)
	properties := tool.InputSchema.Properties

	requiredParams := []string{"from_token", "to_token", "amount", "slippage_tolerance", "user_address"}
	for _, param := range requiredParams {
		_, exists := properties[param]
		suite.True(exists, "Parameter %s should exist", param)
	}
}

func (suite *SwapTokensTestSuite) TestSwapETHForTokens() {
	suite.T().Log("Testing ETH to Token swap...")

	// Get initial balances
	initialETH, err := suite.ethClient.BalanceAt(context.Background(), suite.testAddress, nil)
	suite.NoError(err)

	var initialTokenBalance *big.Int
	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&initialTokenBalance}, "balanceOf", suite.testAddress)
	suite.NoError(err)

	// Create swap request
	swapAmount := big.NewInt(0).Mul(big.NewInt(1), big.NewInt(1e18)) // 1 ETH
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"from_token":         services.EthTokenAddress,
				"to_token":           suite.testToken.Address.Hex(),
				"amount":             swapAmount.String(),
				"slippage_tolerance": "1.0",
				"user_address":       suite.testAddress.Hex(),
			},
		},
	}

	// Execute swap through tool
	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)

	// Log error if present for debugging
	if result.IsError && len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.T().Logf("Error from tool: %s", textContent.Text)
		}
	}

	suite.False(result.IsError)

	// Extract session ID
	var sessionIDContent string
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		sessionIDContent = textContent.Text
	}
	sessionID := strings.TrimPrefix(sessionIDContent, "Swap transaction session created: ")

	// Get transaction session and deployments
	session, err := suite.txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.Len(session.TransactionDeployments, 1) // ETH to Token has only 1 transaction

	// Execute the swap transaction
	suite.T().Log("Executing swap transaction...")
	deployment := session.TransactionDeployments[0]
	txReceipt, err := suite.executeTransaction(deployment.Data, deployment.Value, deployment.Receiver)
	suite.NoError(err)
	suite.Equal(uint64(1), txReceipt.Status, "Swap transaction should succeed")

	// Verify balances changed
	finalETH, err := suite.ethClient.BalanceAt(context.Background(), suite.testAddress, nil)
	suite.NoError(err)

	var finalTokenBalance *big.Int
	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&finalTokenBalance}, "balanceOf", suite.testAddress)
	suite.NoError(err)

	// ETH should decrease (more than swap amount due to gas)
	suite.True(initialETH.Cmp(finalETH) > 0, "ETH balance should decrease")

	// Token balance should increase
	suite.True(finalTokenBalance.Cmp(initialTokenBalance) > 0, "Token balance should increase")

	suite.T().Logf("✓ Successfully swapped ETH for tokens")
	suite.T().Logf("  ETH spent: %s wei", new(big.Int).Sub(initialETH, finalETH).String())
	suite.T().Logf("  Tokens received: %s", new(big.Int).Sub(finalTokenBalance, initialTokenBalance).String())
}

func (suite *SwapTokensTestSuite) TestSwapTokensForETH() {
	suite.T().Log("Testing Token to ETH swap...")

	// Get initial balances
	initialETH, err := suite.ethClient.BalanceAt(context.Background(), suite.testAddress, nil)
	suite.NoError(err)

	var initialTokenBalance *big.Int
	err = suite.testToken2.BoundContract.Call(nil, &[]interface{}{&initialTokenBalance}, "balanceOf", suite.testAddress)
	suite.NoError(err)

	// Create swap request
	swapAmount := big.NewInt(0).Mul(big.NewInt(1000), big.NewInt(1e18)) // 1000 tokens
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"from_token":         suite.testToken2.Address.Hex(),
				"to_token":           services.EthTokenAddress,
				"amount":             swapAmount.String(),
				"slippage_tolerance": "1.0",
				"user_address":       suite.testAddress.Hex(),
			},
		},
	}

	// Execute swap through tool
	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)

	// Extract session ID
	var sessionIDContent string
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		sessionIDContent = textContent.Text
	}
	sessionID := strings.TrimPrefix(sessionIDContent, "Swap transaction session created: ")

	// Get transaction session and deployments
	session, err := suite.txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.Len(session.TransactionDeployments, 2) // Token to ETH has 2 transactions (approve + swap)

	// Execute transactions in order
	suite.T().Log("Executing swap transactions...")
	for i, deployment := range session.TransactionDeployments {
		suite.T().Logf("Executing transaction %d: %s", i+1, deployment.Title)

		txReceipt, err := suite.executeTransaction(deployment.Data, deployment.Value, deployment.Receiver)
		suite.NoError(err)
		suite.Equal(uint64(1), txReceipt.Status, "Transaction %d should succeed", i+1)
	}

	// Verify balances changed
	finalETH, err := suite.ethClient.BalanceAt(context.Background(), suite.testAddress, nil)
	suite.NoError(err)

	var finalTokenBalance *big.Int
	err = suite.testToken2.BoundContract.Call(nil, &[]interface{}{&finalTokenBalance}, "balanceOf", suite.testAddress)
	suite.NoError(err)

	// ETH should increase (minus gas costs)
	suite.True(finalETH.Cmp(initialETH) > 0 ||
		new(big.Int).Sub(initialETH, finalETH).Cmp(big.NewInt(1e17)) < 0, // Allow for gas costs
		"ETH balance should increase or gas cost should be reasonable")

	// Token balance should decrease
	suite.True(initialTokenBalance.Cmp(finalTokenBalance) > 0, "Token balance should decrease")

	suite.T().Logf("✓ Successfully swapped tokens for ETH")
	suite.T().Logf("  Tokens spent: %s", new(big.Int).Sub(initialTokenBalance, finalTokenBalance).String())
	suite.T().Logf("  ETH received (approx): %s wei", new(big.Int).Sub(finalETH, initialETH).String())
}

func (suite *SwapTokensTestSuite) TestSwapTokensForTokens() {
	suite.T().Log("Testing Token to Token swap (via WETH)...")

	// Get initial balances
	var initialToken1Balance *big.Int
	err := suite.testToken.BoundContract.Call(nil, &[]interface{}{&initialToken1Balance}, "balanceOf", suite.testAddress)
	suite.NoError(err)

	var initialToken2Balance *big.Int
	err = suite.testToken2.BoundContract.Call(nil, &[]interface{}{&initialToken2Balance}, "balanceOf", suite.testAddress)
	suite.NoError(err)

	// Create swap request
	swapAmount := big.NewInt(0).Mul(big.NewInt(100), big.NewInt(1e18)) // 100 tokens
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"from_token":         suite.testToken.Address.Hex(),
				"to_token":           suite.testToken2.Address.Hex(),
				"amount":             swapAmount.String(),
				"slippage_tolerance": "2.0", // Higher slippage for multi-hop
				"user_address":       suite.testAddress.Hex(),
			},
		},
	}

	// Execute swap through tool
	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)

	// Extract session ID
	var sessionIDContent string
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		sessionIDContent = textContent.Text
	}
	sessionID := strings.TrimPrefix(sessionIDContent, "Swap transaction session created: ")

	// Get transaction session and deployments
	session, err := suite.txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.Len(session.TransactionDeployments, 2) // Token to Token has 2 transactions (approve + swap)

	// Execute transactions in order
	suite.T().Log("Executing swap transactions...")
	for i, deployment := range session.TransactionDeployments {
		suite.T().Logf("Executing transaction %d: %s", i+1, deployment.Title)

		txReceipt, err := suite.executeTransaction(deployment.Data, deployment.Value, deployment.Receiver)
		suite.NoError(err)
		suite.Equal(uint64(1), txReceipt.Status, "Transaction %d should succeed", i+1)
	}

	// Verify balances changed
	var finalToken1Balance *big.Int
	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&finalToken1Balance}, "balanceOf", suite.testAddress)
	suite.NoError(err)

	var finalToken2Balance *big.Int
	err = suite.testToken2.BoundContract.Call(nil, &[]interface{}{&finalToken2Balance}, "balanceOf", suite.testAddress)
	suite.NoError(err)

	// Token1 balance should decrease
	suite.True(initialToken1Balance.Cmp(finalToken1Balance) > 0, "Token1 balance should decrease")

	// Token2 balance should increase
	suite.True(finalToken2Balance.Cmp(initialToken2Balance) > 0, "Token2 balance should increase")

	suite.T().Logf("✓ Successfully swapped Token1 for Token2 (via WETH)")
	suite.T().Logf("  Token1 spent: %s", new(big.Int).Sub(initialToken1Balance, finalToken1Balance).String())
	suite.T().Logf("  Token2 received: %s", new(big.Int).Sub(finalToken2Balance, initialToken2Balance).String())
}

func (suite *SwapTokensTestSuite) TestInvalidSwapParameters() {
	// Test invalid slippage
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"from_token":         services.EthTokenAddress,
				"to_token":           suite.testToken.Address.Hex(),
				"amount":             "1000000000000000000",
				"slippage_tolerance": "101", // Invalid: > 100%
				"user_address":       suite.testAddress.Hex(),
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "slippage")
	}
}

func (suite *SwapTokensTestSuite) TestSwapSameToken() {
	// Test swapping token to itself
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"from_token":         suite.testToken.Address.Hex(),
				"to_token":           suite.testToken.Address.Hex(),
				"amount":             "1000000000000000000",
				"slippage_tolerance": "1.0",
				"user_address":       suite.testAddress.Hex(),
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Cannot swap token to itself")
	}
}

func TestSwapTokensTestSuite(t *testing.T) {
	// Check if Ethereum testnet is available before running tests
	client, err := ethclient.Dial(SWAP_TEST_TESTNET_RPC)
	if err != nil {
		t.Skipf("Skipping swap tokens tests: Ethereum testnet not available at %s. Run 'make e2e-network' to start testnet.", SWAP_TEST_TESTNET_RPC)
		return
	}

	// Verify network connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	networkID, err := client.NetworkID(ctx)
	client.Close()

	if err != nil || networkID.Cmp(big.NewInt(31337)) != 0 {
		t.Skipf("Skipping swap tokens tests: Cannot connect to anvil testnet at %s (network ID should be 31337). Run 'make e2e-network' to start testnet.", SWAP_TEST_TESTNET_RPC)
		return
	}

	suite.Run(t, new(SwapTokensTestSuite))
}
