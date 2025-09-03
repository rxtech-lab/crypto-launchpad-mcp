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
	db                services.DBService
	ethClient         *ethclient.Client
	tool              *createLiquidityPoolTool
	chain             *models.Chain
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
	chainService     services.ChainService
}

func (suite *CreateLiquidityPoolTestSuite) SetupSuite() {
	// Initialize in-memory database
	db, err := services.NewSqliteDBService(":memory:")
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
	suite.txService = services.NewTransactionService(db.GetDB())
	suite.liquidityService = services.NewLiquidityService(db.GetDB())
	suite.uniswapService = services.NewUniswapService(db.GetDB())
	suite.chainService = services.NewChainService(db.GetDB())

	// Initialize tool
	suite.tool = NewCreateLiquidityPoolTool(
		suite.chainService,
		POOL_TEST_SERVER_PORT,
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

	// Also reset any pair state by clearing existing pairs (not directly possible)
	// Note: In a real test environment, we would reset the blockchain state
	// For now, we clean up database state which is sufficient for most tests
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

	err := suite.chainService.CreateChain(chain)
	suite.Require().NoError(err)
	suite.chain = chain
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

func (suite *CreateLiquidityPoolTestSuite) cleanupTestData() {
	// Clean up transaction sessions
	suite.db.GetDB().Where("1 = 1").Delete(&models.TransactionSession{})

	// Clean up liquidity pools
	suite.db.GetDB().Where("1 = 1").Delete(&models.LiquidityPool{})

}

// Test cases

func (suite *CreateLiquidityPoolTestSuite) TestGetTool() {
	tool := suite.tool.GetTool()

	suite.Equal("create_liquidity_pool", tool.Name)
	suite.Contains(tool.Description, "Create new Uniswap liquidity pool")
	suite.Contains(tool.Description, "ETH-to-Token pairs")
	suite.Contains(tool.Description, "Token-to-Token pairs")

	// Check required parameters exist
	suite.NotNil(tool.InputSchema)
	properties := tool.InputSchema.Properties

	requiredParams := []string{"token0_address", "token1_address", "initial_token0_amount", "initial_token1_amount", "owner_address"}
	for _, param := range requiredParams {
		_, exists := properties[param]
		suite.True(exists, "Parameter %s should exist", param)
	}
}

// Create a liquidity pool to swap between ETH and a token
func (suite *CreateLiquidityPoolTestSuite) TestCreateLiquidityPoolWithEth() {
	// Create liquidity pool via tool (ETH pair)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token0_address":        suite.testToken.Address.Hex(),
				"token1_address":        ETH_TOKEN_ADDRESS,
				"initial_token0_amount": "1000000000000000000000", // 1000 tokens
				"initial_token1_amount": "500000000000000000",     // 0.5 ETH
				"owner_address":         suite.testAddress.Hex(),
			},
		},
	}

	// check the balance of the test token
	var tokenBalance1 *big.Int
	err := suite.testToken.BoundContract.Call(nil, &[]interface{}{&tokenBalance1}, "balanceOf", suite.testAddress)
	suite.NoError(err)
	suite.True(tokenBalance1.Cmp(big.NewInt(0)) > 0, "Test token balance should be > 0")
	suite.T().Logf("✓ Test token balance: %s", tokenBalance1.String())

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
	sessionID := strings.TrimPrefix(sessionIDContent, "Transaction session created: ")

	// Get transaction session and deployments
	session, err := suite.txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.Len(session.TransactionDeployments, 2) // ETH pairs have 2 transactions

	// Execute transactions in order
	suite.T().Log("Executing blockchain transactions...")

	// Execute each transaction
	for i, deployment := range session.TransactionDeployments {
		suite.T().Logf("Executing transaction %d: %s", i+1, deployment.Title)

		txReceipt, err := suite.executeTransaction(deployment.Data, deployment.Value, deployment.Receiver)
		suite.NoError(err)
		suite.Equal(uint64(1), txReceipt.Status, "Transaction %d should succeed", i+1)
		suite.T().Logf("✓ Transaction %d successful", i+1)
	}

	// Check if pair was created
	var pairAddress common.Address
	err = suite.factoryContract.BoundContract.Call(nil, &[]any{&pairAddress}, "getPair", suite.testToken.Address, suite.weth9Contract.Address)
	suite.NoError(err)
	suite.NotEqual(common.Address{}, pairAddress, "Pair should be created")
	suite.T().Logf("✓ Pair created at: %s", pairAddress.Hex())

	// Verify liquidity pool has correct token balances
	suite.T().Log("Verifying liquidity pool balances...")

	// Check token balance in the pair contract
	var tokenBalance *big.Int
	err = suite.testToken.BoundContract.Call(nil, &[]any{&tokenBalance}, "balanceOf", pairAddress)
	suite.NoError(err)
	expectedTokenAmount := new(big.Int)
	expectedTokenAmount.SetString("1000000000000000000000", 10) // 1000 tokens
	suite.Equal(expectedTokenAmount, tokenBalance, "Pair should have correct token balance")
	suite.T().Logf("✓ Token balance in pair: %s (expected: %s)", tokenBalance.String(), expectedTokenAmount.String())

	// Verify LP tokens were minted to the owner
	// We need to create a bound contract for the pair to check LP balance
	pairABI := `[{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"}]`
	parsedPairABI, err := abi.JSON(strings.NewReader(pairABI))
	suite.NoError(err)
	pairContract := bind.NewBoundContract(pairAddress, parsedPairABI, suite.ethClient, suite.ethClient, suite.ethClient)

	// Check LP token balance of owner
	var lpBalance *big.Int
	err = pairContract.Call(nil, &[]interface{}{&lpBalance}, "balanceOf", suite.testAddress)
	suite.NoError(err)
	suite.True(lpBalance.Cmp(big.NewInt(0)) > 0, "Owner should have LP tokens")
	suite.T().Logf("✓ LP token balance for owner: %s", lpBalance.String())

	// Check total LP supply
	var totalLPSupply *big.Int
	err = pairContract.Call(nil, &[]interface{}{&totalLPSupply}, "totalSupply")
	suite.NoError(err)
	suite.True(totalLPSupply.Cmp(big.NewInt(0)) > 0, "Total LP supply should be > 0")
	suite.T().Logf("✓ Total LP token supply: %s", totalLPSupply.String())

	suite.T().Log("✓ All blockchain operations and balance verifications completed successfully")
}

// executeTransaction is a helper method to execute a transaction on the blockchain
func (suite *CreateLiquidityPoolTestSuite) executeTransaction(data, value, to string) (*types.Receipt, error) {
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
	privateKey, err := crypto.HexToECDSA(POOL_TEST_PRIVATE_KEY)
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
