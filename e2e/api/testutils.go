package e2e

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rxtech-lab/launchpad-mcp/internal/api"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/require"
)

const (
	// Ethereum testnet configuration
	TESTNET_RPC      = "http://localhost:8545"
	TESTNET_CHAIN_ID = "31337" // Anvil default
)

// TestSetup holds all test infrastructure
type TestSetup struct {
	DB         *database.Database
	APIServer  *api.APIServer
	ServerPort int
	EthClient  *ethclient.Client
	TestKeys   []*TestAccount
	TempDir    string
	t          *testing.T
}

// TestAccount represents a test Ethereum account
type TestAccount struct {
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
	Auth       *bind.TransactOpts
}

// NewTestSetup creates a complete test environment
func NewTestSetup(t *testing.T) *TestSetup {
	setup := &TestSetup{t: t}

	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "launchpad-test-*")
	require.NoError(t, err)
	setup.TempDir = tempDir

	// Initialize test database (use file database to ensure foreign keys work)
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.NewDatabase(dbPath)
	require.NoError(t, err)
	setup.DB = db

	// Initialize API server
	apiServer := api.NewAPIServer(db)
	port, err := apiServer.Start()
	require.NoError(t, err)
	setup.APIServer = apiServer
	setup.ServerPort = port

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Initialize Ethereum client
	ethClient, err := ethclient.Dial(TESTNET_RPC)
	require.NoError(t, err)
	setup.EthClient = ethClient

	// Initialize test accounts
	setup.initTestAccounts()

	// Setup default chain configuration
	setup.setupDefaultChain()

	return setup
}

// initTestAccounts creates test accounts from predefined private keys
func (s *TestSetup) initTestAccounts() {
	privateKeys := []string{
		TESTING_PK_1,
		TESTING_PK_2,
	}

	for _, pkHex := range privateKeys {
		// Remove 0x prefix if present
		pkHex = strings.TrimPrefix(pkHex, "0x")

		privateKey, err := crypto.HexToECDSA(pkHex)
		require.NoError(s.t, err)

		address := crypto.PubkeyToAddress(privateKey.PublicKey)

		// Create transaction options
		chainID := big.NewInt(31337) // Anvil default
		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
		require.NoError(s.t, err)

		// Set reasonable gas limit and price for tests
		auth.GasLimit = 5000000
		auth.GasPrice = big.NewInt(1000000000) // 1 gwei

		testAccount := &TestAccount{
			PrivateKey: privateKey,
			Address:    address,
			Auth:       auth,
		}

		s.TestKeys = append(s.TestKeys, testAccount)
	}
}

// setupDefaultChain configures the default Ethereum testnet chain
func (s *TestSetup) setupDefaultChain() {
	chain := &models.Chain{
		ChainType: "ethereum",
		RPC:       TESTNET_RPC,
		ChainID:   TESTNET_CHAIN_ID,
		Name:      "Ethereum Testnet",
		IsActive:  true,
	}

	err := s.DB.CreateChain(chain)
	require.NoError(s.t, err)
}

// CreateTestTemplate creates a test template in the database
func (s *TestSetup) CreateTestTemplate(name, description, contractCode string) *models.Template {
	template := &models.Template{
		Name:         name,
		Description:  description,
		ChainType:    "ethereum",
		TemplateCode: contractCode,
	}

	err := s.DB.CreateTemplate(template)
	require.NoError(s.t, err)

	return template
}

// GetTestChainID returns the ID of the default test chain
func (s *TestSetup) GetTestChainID() uint {
	chain, err := s.DB.GetActiveChain()
	require.NoError(s.t, err)
	return chain.ID
}

// GetPrimaryTestAccount returns the first test account
func (s *TestSetup) GetPrimaryTestAccount() *TestAccount {
	require.Greater(s.t, len(s.TestKeys), 0, "No test accounts available")
	return s.TestKeys[0]
}

// GetSecondaryTestAccount returns the second test account
func (s *TestSetup) GetSecondaryTestAccount() *TestAccount {
	require.Greater(s.t, len(s.TestKeys), 1, "Secondary test account not available")
	return s.TestKeys[1]
}

// MakeAPIRequest makes an HTTP request to the test API server
func (s *TestSetup) MakeAPIRequest(method, path string) (*http.Response, error) {
	url := fmt.Sprintf("http://localhost:%d%s", s.ServerPort, path)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

// VerifyEthereumConnection checks that the Ethereum testnet is accessible
func (s *TestSetup) VerifyEthereumConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check network ID
	networkID, err := s.EthClient.NetworkID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get network ID: %w", err)
	}

	if networkID.Cmp(big.NewInt(31337)) != 0 {
		return fmt.Errorf("unexpected network ID: got %s, expected 31337", networkID.String())
	}

	// Check that test accounts have some ETH
	for i, account := range s.TestKeys {
		balance, err := s.EthClient.BalanceAt(ctx, account.Address, nil)
		if err != nil {
			return fmt.Errorf("failed to get balance for account %d: %w", i, err)
		}

		if balance.Cmp(big.NewInt(0)) == 0 {
			return fmt.Errorf("account %d (%s) has zero balance", i, account.Address.Hex())
		}
	}

	return nil
}

// WaitForTransaction waits for a transaction to be mined
func (s *TestSetup) WaitForTransaction(txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for transaction %s", txHash.Hex())
		default:
			receipt, err := s.EthClient.TransactionReceipt(ctx, txHash)
			if err == nil {
				if receipt.Status == 1 {
					return receipt, nil
				}
				return receipt, fmt.Errorf("transaction %s failed", txHash.Hex())
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Cleanup properly shuts down all test infrastructure
func (s *TestSetup) Cleanup() {
	if s.APIServer != nil {
		s.APIServer.Shutdown()
	}
	if s.DB != nil {
		s.DB.Close()
	}
	if s.EthClient != nil {
		s.EthClient.Close()
	}
	if s.TempDir != "" {
		os.RemoveAll(s.TempDir)
	}
}

// AssertServerHealth checks that the API server is responding
func (s *TestSetup) AssertServerHealth() {
	resp, err := s.MakeAPIRequest("GET", "/health")
	require.NoError(s.t, err)
	defer resp.Body.Close()

	require.Equal(s.t, http.StatusOK, resp.StatusCode)
}

// DeployContractResult holds the result of a contract deployment
type DeployContractResult struct {
	ContractAddress common.Address
	TransactionHash common.Hash
	Receipt         *types.Receipt
	ABI             abi.ABI
	BoundContract   *bind.BoundContract
}

// DeployContract compiles and deploys a Solidity contract to the testnet
func (s *TestSetup) DeployContract(account *TestAccount, contractCode, contractName string, constructorArgs ...interface{}) (*DeployContractResult, error) {
	ctx := context.Background()

	// Compile the Solidity contract
	compilationResult, err := utils.CompileSolidity("0.8.19", contractCode)
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	if _, exists := compilationResult.Bytecode[contractName]; !exists {
		return nil, fmt.Errorf("contract %s not found in compilation result", contractName)
	}

	// Get bytecode and ABI for deployment
	bytecodeHex := compilationResult.Bytecode[contractName]
	bytecode, err := hex.DecodeString(strings.TrimPrefix(bytecodeHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode bytecode: %w", err)
	}

	// Parse ABI - the compilation result returns ABI as an array, not object
	abiData := compilationResult.Abi[contractName]
	abiBytes, err := json.Marshal(abiData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ABI: %w", err)
	}
	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Pack constructor arguments if provided
	var constructorData []byte
	if len(constructorArgs) > 0 {
		constructorData, err = parsedABI.Pack("", constructorArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to pack constructor arguments: %w", err)
		}
	}

	// Combine bytecode with constructor arguments
	fullBytecode := append(bytecode, constructorData...)

	// Create deployment transaction
	nonce, err := s.EthClient.PendingNonceAt(ctx, account.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := s.EthClient.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Estimate gas for deployment (generous limit)
	gasLimit := uint64(5000000)

	tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, gasPrice, fullBytecode)

	// Sign and send transaction
	chainID := big.NewInt(31337)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), account.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = s.EthClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(ctx, s.EthClient, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction: %w", err)
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("transaction failed with status %d", receipt.Status)
	}

	// Create bound contract for easy interaction
	boundContract := bind.NewBoundContract(receipt.ContractAddress, parsedABI, s.EthClient, s.EthClient, s.EthClient)

	return &DeployContractResult{
		ContractAddress: receipt.ContractAddress,
		TransactionHash: signedTx.Hash(),
		Receipt:         receipt,
		ABI:             parsedABI,
		BoundContract:   boundContract,
	}, nil
}

// CallContractView calls a view function on a deployed contract
func (s *TestSetup) CallContractView(result *DeployContractResult, functionName string, args ...interface{}) ([]interface{}, error) {
	var output []interface{}
	err := result.BoundContract.Call(&bind.CallOpts{}, &output, functionName, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to call %s: %w", functionName, err)
	}
	return output, nil
}

// CallContractTx sends a transaction to a deployed contract
func (s *TestSetup) CallContractTx(account *TestAccount, result *DeployContractResult, functionName string, args ...interface{}) (*types.Receipt, error) {
	tx, err := result.BoundContract.Transact(account.Auth, functionName, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction to %s: %w", functionName, err)
	}

	ctx := context.Background()
	receipt, err := bind.WaitMined(ctx, s.EthClient, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction: %w", err)
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("transaction failed with status %d", receipt.Status)
	}

	return receipt, nil
}
