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
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/suite"
)

const (
	ADD_LIQ_TEST_TESTNET_RPC      = "http://localhost:8545"
	ADD_LIQ_TEST_TESTNET_CHAIN_ID = "31337"
	ADD_LIQ_TEST_SERVER_PORT      = 9998
	// Test private key for Anvil (account #0)
	ADD_LIQ_TEST_PRIVATE_KEY = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

// DeployedContract represents a deployed contract with its details
type AddLiquidityDeployedContract struct {
	Address         common.Address
	TransactionHash common.Hash
	ABI             abi.ABI
	BoundContract   *bind.BoundContract
}

type AddLiquidityTestSuite struct {
	suite.Suite
	db                services.DBService
	ethClient         *ethclient.Client
	tool              *addLiquidityTool
	chain             *models.Chain
	uniswapDeployment *models.UniswapDeployment
	testAccount       *bind.TransactOpts
	testAddress       common.Address
	liquidityPool     *models.LiquidityPool

	// Deployed contracts
	testToken       *AddLiquidityDeployedContract
	weth9Contract   *AddLiquidityDeployedContract
	factoryContract *AddLiquidityDeployedContract
	routerContract  *AddLiquidityDeployedContract

	// Services
	evmService       services.EvmService
	txService        services.TransactionService
	liquidityService services.LiquidityService
	uniswapService   services.UniswapService
	chainService     services.ChainService
}

func (suite *AddLiquidityTestSuite) SetupSuite() {
	// Initialize in-memory database
	db, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Initialize Ethereum client
	ethClient, err := ethclient.Dial(ADD_LIQ_TEST_TESTNET_RPC)
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
	suite.tool = NewAddLiquidityTool(
		suite.chainService,
		ADD_LIQ_TEST_SERVER_PORT,
		suite.evmService,
		suite.txService,
		suite.liquidityService,
		suite.uniswapService,
	)

	// Setup test data
	suite.setupTestChain()

	// Deploy contracts
	suite.deployContracts()

	// Setup Uniswap deployment
	suite.setupUniswapDeployment()

	// Create initial liquidity pool
	suite.createInitialLiquidityPool()
}

func (suite *AddLiquidityTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.ethClient != nil {
		suite.ethClient.Close()
	}
}

func (suite *AddLiquidityTestSuite) SetupTest() {
	// Clean up any existing sessions and positions for each test
	suite.cleanupTestData()
}

func (suite *AddLiquidityTestSuite) verifyEthereumConnection() error {
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

func (suite *AddLiquidityTestSuite) setupTestAccount() {
	privateKey, err := crypto.HexToECDSA(ADD_LIQ_TEST_PRIVATE_KEY)
	suite.Require().NoError(err)

	suite.testAddress = crypto.PubkeyToAddress(privateKey.PublicKey)

	chainID := big.NewInt(31337)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	suite.Require().NoError(err)

	auth.GasLimit = 5000000
	auth.GasPrice = big.NewInt(1000000000) // 1 gwei

	suite.testAccount = auth
}

func (suite *AddLiquidityTestSuite) setupTestChain() {
	chain := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		RPC:       ADD_LIQ_TEST_TESTNET_RPC,
		NetworkID: ADD_LIQ_TEST_TESTNET_CHAIN_ID,
		Name:      "Ethereum Testnet",
		IsActive:  true,
	}

	err := suite.chainService.CreateChain(chain)
	suite.Require().NoError(err)
	suite.chain = chain
}

func (suite *AddLiquidityTestSuite) deployContracts() {
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

func (suite *AddLiquidityTestSuite) deployOpenZeppelinToken(name, symbol string, supply *big.Int) (*AddLiquidityDeployedContract, error) {
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

func (suite *AddLiquidityTestSuite) deployWETH9() (*AddLiquidityDeployedContract, error) {
	// Get WETH9 artifact from embedded contracts
	artifact, err := contracts.GetWETH9Artifact()
	if err != nil {
		return nil, fmt.Errorf("failed to get WETH9 artifact: %w", err)
	}

	// Deploy the contract
	return suite.deployContract(artifact.Bytecode, artifact.ABI)
}

func (suite *AddLiquidityTestSuite) deployUniswapV2Factory() (*AddLiquidityDeployedContract, error) {
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

func (suite *AddLiquidityTestSuite) deployUniswapV2Router(factoryAddress, wethAddress common.Address) (*AddLiquidityDeployedContract, error) {
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

func (suite *AddLiquidityTestSuite) deployContract(bytecodeHex string, abiData interface{}) (*AddLiquidityDeployedContract, error) {
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

func (suite *AddLiquidityTestSuite) deployContractRaw(bytecodeHex string, abiData interface{}, parsedABI abi.ABI) (*AddLiquidityDeployedContract, error) {
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
	privateKey, err := crypto.HexToECDSA(ADD_LIQ_TEST_PRIVATE_KEY)
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

	return &AddLiquidityDeployedContract{
		Address:         receipt.ContractAddress,
		TransactionHash: signedTx.Hash(),
		ABI:             parsedABI,
		BoundContract:   boundContract,
	}, nil
}

func (suite *AddLiquidityTestSuite) waitForTransaction(txHash common.Hash) (*types.Receipt, error) {
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

func (suite *AddLiquidityTestSuite) setupUniswapDeployment() {
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

func (suite *AddLiquidityTestSuite) createInitialLiquidityPool() {
	suite.T().Log("Creating initial liquidity pool...")

	// First, approve tokens and add initial liquidity on-chain
	suite.setupInitialLiquidityOnChain()

	// Create liquidity pool record with confirmed status
	pool := &models.LiquidityPool{
		TokenAddress:   suite.testToken.Address.Hex(),
		UniswapVersion: "v2",
		Token0:         suite.testToken.Address.Hex(),
		Token1:         suite.weth9Contract.Address.Hex(),
		InitialToken0:  "100000000000000000000", // 100 tokens
		InitialToken1:  "1000000000000000000",   // 1 ETH
		CreatorAddress: suite.testAddress.Hex(),
		Status:         models.TransactionStatusConfirmed,
	}

	// Get the pair address from the factory
	var pairAddress common.Address
	err := suite.factoryContract.BoundContract.Call(nil, &[]interface{}{&pairAddress}, "getPair", suite.testToken.Address, suite.weth9Contract.Address)
	suite.Require().NoError(err)
	suite.Require().NotEqual(common.Address{}, pairAddress, "Pair should exist after initial liquidity setup")

	pool.PairAddress = pairAddress.Hex()

	poolID, err := suite.liquidityService.CreateLiquidityPool(pool)
	suite.Require().NoError(err)

	suite.liquidityPool, err = suite.liquidityService.GetLiquidityPool(poolID)
	suite.Require().NoError(err)
	suite.T().Logf("✓ Created initial liquidity pool with ID %d", poolID)
}

func (suite *AddLiquidityTestSuite) setupInitialLiquidityOnChain() {
	suite.T().Log("Setting up initial liquidity on-chain...")

	// Check initial balances
	balance, err := suite.ethClient.BalanceAt(context.Background(), suite.testAddress, nil)
	suite.Require().NoError(err)
	suite.T().Logf("Initial ETH balance: %s", balance.String())

	var tokenBalance *big.Int
	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&tokenBalance}, "balanceOf", suite.testAddress)
	suite.Require().NoError(err)
	suite.T().Logf("Initial token balance: %s", tokenBalance.String())

	// Deposit ETH to get WETH
	auth := *suite.testAccount
	auth.Value = big.NewInt(2e18) // 2 ETH
	suite.T().Log("Depositing ETH to WETH...")
	tx, err := suite.weth9Contract.BoundContract.Transact(&auth, "deposit")
	suite.Require().NoError(err)
	receipt, err := suite.waitForTransaction(tx.Hash())
	suite.Require().NoError(err)
	if receipt.Status != 1 {
		suite.T().Logf("WETH deposit transaction failed. Receipt: %+v", receipt)
	}
	suite.Require().Equal(uint64(1), receipt.Status)

	// Approve tokens for router
	maxUint256 := new(big.Int)
	maxUint256.SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)

	// Approve test token
	auth.Value = big.NewInt(0)
	suite.T().Log("Approving test token for router...")
	tx, err = suite.testToken.BoundContract.Transact(&auth, "approve", suite.routerContract.Address, maxUint256)
	suite.Require().NoError(err)
	receipt, err = suite.waitForTransaction(tx.Hash())
	suite.Require().NoError(err)
	if receipt.Status != 1 {
		suite.T().Logf("Token approval transaction failed. Receipt: %+v", receipt)
	}
	suite.Require().Equal(uint64(1), receipt.Status)

	// Approve WETH
	suite.T().Log("Approving WETH for router...")
	tx, err = suite.weth9Contract.BoundContract.Transact(&auth, "approve", suite.routerContract.Address, maxUint256)
	suite.Require().NoError(err)
	receipt, err = suite.waitForTransaction(tx.Hash())
	suite.Require().NoError(err)
	if receipt.Status != 1 {
		suite.T().Logf("WETH approval transaction failed. Receipt: %+v", receipt)
	}
	suite.Require().Equal(uint64(1), receipt.Status)

	// Add initial liquidity via router
	tokenAmount := new(big.Int)
	tokenAmount.SetString("100000000000000000000", 10) // 100 tokens
	wethAmount := new(big.Int)
	wethAmount.SetString("1000000000000000000", 10) // 1 ETH (we deposited 2 ETH, so this is safe)
	minTokenAmount := new(big.Int)
	minTokenAmount.SetString("99000000000000000000", 10) // 99 tokens (1% slippage)
	minWethAmount := new(big.Int)
	minWethAmount.SetString("990000000000000000", 10) // 0.99 ETH (1% slippage)
	deadline := time.Now().Unix() + 600

	// Check WETH balance before adding liquidity
	var wethBalance *big.Int
	err = suite.weth9Contract.BoundContract.Call(nil, &[]interface{}{&wethBalance}, "balanceOf", suite.testAddress)
	suite.Require().NoError(err)
	suite.T().Logf("WETH balance before addLiquidity: %s", wethBalance.String())

	// Verify allowances are set correctly
	var tokenAllowance, wethAllowance *big.Int
	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&tokenAllowance}, "allowance", suite.testAddress, suite.routerContract.Address)
	suite.Require().NoError(err)
	err = suite.weth9Contract.BoundContract.Call(nil, &[]interface{}{&wethAllowance}, "allowance", suite.testAddress, suite.routerContract.Address)
	suite.Require().NoError(err)
	suite.T().Logf("Token allowance: %s, WETH allowance: %s", tokenAllowance.String(), wethAllowance.String())

	// Ensure we have sufficient allowances
	suite.True(tokenAllowance.Cmp(tokenAmount) >= 0, "Token allowance should be >= token amount")
	suite.True(wethAllowance.Cmp(wethAmount) >= 0, "WETH allowance should be >= WETH amount")

	suite.T().Log("Adding initial liquidity using addLiquidityETH...")
	// Use addLiquidityETH instead - this is more appropriate for ETH pairs
	// Reset auth value to send ETH
	auth.Value = wethAmount // Send ETH directly
	tx, err = suite.routerContract.BoundContract.Transact(&auth, "addLiquidityETH",
		suite.testToken.Address, // token
		tokenAmount,             // amountTokenDesired
		minTokenAmount,          // amountTokenMin
		minWethAmount,           // amountETHMin
		suite.testAddress,       // to
		big.NewInt(deadline),    // deadline
	)
	suite.Require().NoError(err)
	receipt, err = suite.waitForTransaction(tx.Hash())
	suite.Require().NoError(err)
	if receipt.Status != 1 {
		suite.T().Logf("Add liquidity transaction failed. Receipt: %+v", receipt)
		suite.T().Logf("Transaction hash: %s", tx.Hash().Hex())
	}
	suite.Require().Equal(uint64(1), receipt.Status)

	suite.T().Log("✓ Initial liquidity setup completed on-chain")
}

func (suite *AddLiquidityTestSuite) cleanupTestData() {
	// Clean up transaction sessions
	suite.db.GetDB().Where("1 = 1").Delete(&models.TransactionSession{})
}

// executeTransaction is a helper method to execute a transaction on the blockchain
func (suite *AddLiquidityTestSuite) executeTransaction(data, value, to string) (*types.Receipt, error) {
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
	privateKey, err := crypto.HexToECDSA(ADD_LIQ_TEST_PRIVATE_KEY)
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

func (suite *AddLiquidityTestSuite) TestGetTool() {
	tool := suite.tool.GetTool()

	suite.Equal("add_liquidity", tool.Name)
	suite.Contains(tool.Description, "Add liquidity to existing Uniswap pool")
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

	// Check token_amount parameter
	tokenAmountProp, exists := properties["token_amount"]
	suite.True(exists)
	if propMap, ok := tokenAmountProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
		suite.Contains(propMap["description"], "Amount of tokens")
	}

	// Check eth_amount parameter
	ethAmountProp, exists := properties["eth_amount"]
	suite.True(exists)
	if propMap, ok := ethAmountProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
		suite.Contains(propMap["description"], "Amount of ETH")
	}

	// Check owner_address parameter
	ownerAddressProp, exists := properties["owner_address"]
	suite.True(exists)
	if propMap, ok := ownerAddressProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
		suite.Contains(propMap["description"], "Address that will receive")
	}
}

func (suite *AddLiquidityTestSuite) TestAddLiquiditySuccess() {
	suite.T().Log("Testing add liquidity success workflow...")

	// Get initial balances before adding liquidity
	var initialTokenBalance, initialWETHBalance, initialLPBalance *big.Int

	pairAddress := common.HexToAddress(suite.liquidityPool.PairAddress)

	err := suite.testToken.BoundContract.Call(nil, &[]interface{}{&initialTokenBalance}, "balanceOf", pairAddress)
	suite.Require().NoError(err)

	err = suite.weth9Contract.BoundContract.Call(nil, &[]interface{}{&initialWETHBalance}, "balanceOf", pairAddress)
	suite.Require().NoError(err)

	// Get initial LP token balance of owner
	pairABI := `[{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"}]`
	parsedPairABI, err := abi.JSON(strings.NewReader(pairABI))
	suite.Require().NoError(err)
	pairContract := bind.NewBoundContract(pairAddress, parsedPairABI, suite.ethClient, suite.ethClient, suite.ethClient)

	err = pairContract.Call(nil, &[]interface{}{&initialLPBalance}, "balanceOf", suite.testAddress)
	suite.Require().NoError(err)

	suite.T().Logf("Initial balances - Token: %s, WETH: %s, LP: %s",
		initialTokenBalance.String(), initialWETHBalance.String(), initialLPBalance.String())

	// Create add liquidity request
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":    suite.testToken.Address.Hex(),
				"token_amount":     "50000000000000000000", // 50 additional tokens
				"eth_amount":       "500000000000000000",   // 0.5 additional ETH
				"min_token_amount": "49000000000000000000", // 49 tokens (2% slippage)
				"min_eth_amount":   "490000000000000000",   // 0.49 ETH (2% slippage)
				"owner_address":    suite.testAddress.Hex(),
				"metadata": []interface{}{
					map[string]interface{}{
						"key":   "Liquidity Action",
						"value": "Add Liquidity",
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
		suite.Contains(textContent.Text, "Please sign the add liquidity transactions")
	}

	if textContent, ok := result.Content[2].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, fmt.Sprintf("http://localhost:%d/tx/", ADD_LIQ_TEST_SERVER_PORT))
	}

	// Extract session ID and verify
	sessionID := strings.TrimPrefix(sessionIDContent, "Transaction session created: ")
	suite.NotEmpty(sessionID)

	// Verify session was created
	session, err := suite.txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.NotNil(session)
	suite.Equal(models.TransactionStatusPending, session.TransactionStatus)
	suite.Len(session.TransactionDeployments, 2) // Token approval + Add liquidity ETH

	// Execute the add liquidity transactions on blockchain
	suite.T().Log("Executing add liquidity transactions on blockchain...")

	// Execute each transaction
	for i, deployment := range session.TransactionDeployments {
		suite.T().Logf("Executing transaction %d: %s", i+1, deployment.Title)

		txReceipt, err := suite.executeTransaction(deployment.Data, deployment.Value, deployment.Receiver)
		suite.NoError(err)
		suite.Equal(uint64(1), txReceipt.Status, "Transaction %d should succeed", i+1)
		suite.T().Logf("✓ Transaction %d successful", i+1)
	}

	// Verify final balances after adding liquidity
	var finalTokenBalance, finalWETHBalance, finalLPBalance *big.Int

	err = suite.testToken.BoundContract.Call(nil, &[]interface{}{&finalTokenBalance}, "balanceOf", pairAddress)
	suite.NoError(err)

	err = suite.weth9Contract.BoundContract.Call(nil, &[]interface{}{&finalWETHBalance}, "balanceOf", pairAddress)
	suite.NoError(err)

	err = pairContract.Call(nil, &[]interface{}{&finalLPBalance}, "balanceOf", suite.testAddress)
	suite.NoError(err)

	suite.T().Logf("Final balances - Token: %s, WETH: %s, LP: %s",
		finalTokenBalance.String(), finalWETHBalance.String(), finalLPBalance.String())

	// Verify balances increased
	expectedTokenIncrease := new(big.Int)
	expectedTokenIncrease.SetString("50000000000000000000", 10)
	actualTokenIncrease := new(big.Int).Sub(finalTokenBalance, initialTokenBalance)
	suite.Equal(expectedTokenIncrease, actualTokenIncrease, "Token balance should increase by exactly 50 tokens")

	expectedWETHIncrease := new(big.Int)
	expectedWETHIncrease.SetString("500000000000000000", 10)
	actualWETHIncrease := new(big.Int).Sub(finalWETHBalance, initialWETHBalance)
	suite.Equal(expectedWETHIncrease, actualWETHIncrease, "WETH balance should increase by exactly 0.5 ETH")

	// Verify LP tokens increased (should be > 0 but exact amount depends on liquidity math)
	lpIncrease := new(big.Int).Sub(finalLPBalance, initialLPBalance)
	suite.True(lpIncrease.Cmp(big.NewInt(0)) > 0, "LP token balance should increase")
	suite.T().Logf("✓ LP token increase: %s", lpIncrease.String())

	suite.T().Log("✓ Add liquidity success test completed successfully")
}

func (suite *AddLiquidityTestSuite) TestToolRegistration() {
	// Test that the tool can be registered with an MCP server
	mcpServer := server.NewMCPServer("test", "1.0.0")

	tool := suite.tool.GetTool()
	handler := suite.tool.GetHandler()

	// This should not panic
	suite.NotPanics(func() {
		mcpServer.AddTool(tool, handler)
	})
}

func TestAddLiquidityTestSuite(t *testing.T) {
	suite.Run(t, new(AddLiquidityTestSuite))
}
