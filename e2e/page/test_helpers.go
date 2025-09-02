package api

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rxtech-lab/launchpad-mcp/internal/api"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/require"
)

// Constants for testing
const (
	TESTNET_RPC      = "http://localhost:8545"
	TESTNET_CHAIN_ID = "31337"
	TESTING_PK_1     = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	TESTING_PK_2     = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
)

// TestSetup provides the core test infrastructure
type TestSetup struct {
	t                 *testing.T
	DBService         services.DBService
	ChainService      services.ChainService
	TemplateService   services.TemplateService
	DeploymentService *services.DeploymentService
	UniswapService    services.UniswapService
	EthClient         *ethclient.Client
	TxService         services.TransactionService
	ServerPort        int
}

// ChromedpTestSetup extends the base TestSetup with chromedp capabilities
type ChromedpTestSetup struct {
	*TestSetup
	apiServer            *api.APIServer
	ctx                  context.Context
	cancel               context.CancelFunc
	walletProviderScript string
}

// NewTestSetup creates the base test infrastructure
func NewTestSetup(t *testing.T) *TestSetup {
	// Create in-memory database service
	dbService, err := services.NewSqliteDBService(":memory:")
	require.NoError(t, err)

	// Create other services
	chainService := services.NewChainService(dbService.GetDB())
	templateService := services.NewTemplateService(dbService.GetDB())
	deploymentService := services.NewDeploymentService(dbService.GetDB())
	uniswapService := services.NewUniswapService(dbService.GetDB())

	// Connect to Ethereum testnet
	ethClient, err := ethclient.Dial(TESTNET_RPC)
	require.NoError(t, err)

	// Create transaction service
	txService := services.NewTransactionService(dbService.GetDB())

	// Get a random port for test server
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	return &TestSetup{
		t:                 t,
		DBService:         dbService,
		ChainService:      chainService,
		TemplateService:   templateService,
		DeploymentService: deploymentService,
		UniswapService:    uniswapService,
		EthClient:         ethClient,
		TxService:         txService,
		ServerPort:        port,
	}
}

// NewChromedpTestSetup creates a complete chromedp test environment
func NewChromedpTestSetup(t *testing.T) *ChromedpTestSetup {
	setup := &ChromedpTestSetup{
		TestSetup: NewTestSetup(t),
	}

	// StartStdioServer HTTP server for E2E tests
	err := setup.startAPIServer()
	require.NoError(t, err)

	// Setup Chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.WindowSize(1280, 720),
	)

	// Create allocator context
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	setup.cancel = cancel

	// Create browser context
	ctx, _ := chromedp.NewContext(allocCtx)
	setup.ctx = ctx

	// Inject wallet provider
	setup.injectWalletProvider()

	return setup
}

// startAPIServer starts the HTTP API server for E2E testing
func (s *ChromedpTestSetup) startAPIServer() error {
	// Initialize hook service
	hookService := services.NewHookService()

	// Initialize API server
	apiServer := api.NewAPIServer(s.TestSetup.DBService, s.TestSetup.TxService, hookService, s.TestSetup.ChainService)
	apiServer.SetupRoutes()
	port, err := apiServer.Start(nil)
	if err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	s.apiServer = apiServer
	s.TestSetup.ServerPort = port

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	return nil
}

// injectWalletProvider injects the test wallet provider script
func (s *ChromedpTestSetup) injectWalletProvider() {
	// Read wallet provider script
	walletProviderPath := filepath.Join(".", "wallet_provider.js")
	walletProviderScript, err := os.ReadFile(walletProviderPath)
	if err != nil {
		// Try relative to current directory
		walletProviderPath = filepath.Join("e2e", "api", "wallet_provider.js")
		walletProviderScript, err = os.ReadFile(walletProviderPath)
		require.NoError(s.t, err, "Failed to read wallet provider script")
	}

	// Add the script to be evaluated on each new page
	s.walletProviderScript = string(walletProviderScript)
}

// SigningRequest represents a request from JavaScript to sign something
type SigningRequest struct {
	Action      string                 `json:"action"`
	PrivateKey  string                 `json:"privateKey"`
	Transaction map[string]interface{} `json:"transaction,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Address     string                 `json:"address,omitempty"`
}

// SigningResponse represents the response to a signing request
type SigningResponse struct {
	Success   bool   `json:"success"`
	TxHash    string `json:"txHash,omitempty"`
	Address   string `json:"address,omitempty"`
	Signature string `json:"signature,omitempty"`
	Error     string `json:"error,omitempty"`
}

// InitializeTestWallet initializes the test wallet in the browser
func (s *ChromedpTestSetup) InitializeTestWallet() error {
	// Get the primary test account private key
	account := s.GetPrimaryTestAccount()
	privateKeyHex := hex.EncodeToString(crypto.FromECDSA(account.PrivateKey))

	// First, add a script to capture and store console messages
	captureScript := `
		window.debugLogs = [];
		const originalLog = console.log;
		const originalError = console.error;
		const originalWarn = console.warn;
		
		console.log = function(...args) {
			window.debugLogs.push({type: 'log', message: args.join(' '), timestamp: Date.now()});
			originalLog.apply(console, args);
		};
		console.error = function(...args) {
			window.debugLogs.push({type: 'error', message: args.join(' '), timestamp: Date.now()});
			originalError.apply(console, args);
		};
		console.warn = function(...args) {
			window.debugLogs.push({type: 'warn', message: args.join(' '), timestamp: Date.now()});
			originalWarn.apply(console, args);
		};
		
		console.log("Debug logging initialized");
	`

	err := chromedp.Run(s.ctx, chromedp.Evaluate(captureScript, nil))
	if err != nil {
		return fmt.Errorf("failed to setup debug logging: %w", err)
	}

	// Inject wallet provider script and initialize wallet
	script := fmt.Sprintf(`
		%s
		
		// Real Go signing function that performs actual blockchain transactions
		window.goSignTransaction = async function(requestJson) {
			const request = JSON.parse(requestJson);
			
			// For testing, we'll use the actual test account address
			if (request.action === "derive_address") {
				return JSON.stringify({
					success: true,
					address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266" // Known anvil address
				});
			} else if (request.action === "sign_transaction") {
				// For real deployment testing, make an actual HTTP request to our test API
				try {
					console.log("Making real transaction request:", request.transaction);
					
					// Get the transaction parameters
					const tx = request.transaction;
					
					// Make HTTP request to our test signing endpoint
					const response = await fetch('/api/test/sign-transaction', {
						method: 'POST',
						headers: {
							'Content-Type': 'application/json'
						},
						body: JSON.stringify({
							privateKey: request.privateKey,
							transaction: tx
						})
					});
					
					if (!response.ok) {
						throw new Error('HTTP ' + response.status + ': ' + await response.text());
					}
					
					const result = await response.json();
					console.log("Transaction signing result:", result);
					
					return JSON.stringify(result);
				} catch (error) {
					console.error("Transaction signing error:", error);
					return JSON.stringify({
						success: false,
						error: error.message
					});
				}
			} else if (request.action === "personal_sign") {
				// For personal sign, we can use a mock since it's not critical for deployment testing
				return JSON.stringify({
					success: true,
					signature: "0x" + Array.from(crypto.getRandomValues(new Uint8Array(65)))
						.map(b => b.toString(16).padStart(2, '0')).join('')
				});
			}
			
			return JSON.stringify({
				success: false,
				error: "Unknown action: " + request.action
			});
		};
		
		console.log("About to initialize test wallet...");
		
		// Initialize test wallet and trigger wallet discovery
		const testProvider = window.initTestWallet("%s", "0x7a69");
		
		console.log("Test wallet initialized, provider:", testProvider);
		
		// Expose test provider globally for console debugging
		window.testProvider = testProvider;
		window.testWallet = window._testWalletProvider;
		
		console.log("Global objects set - testProvider:", !!window.testProvider, "testWallet:", !!window.testWallet);
		
		// Force wallet discovery after a short delay to ensure everything is ready
		setTimeout(() => {
			console.log("Forcing wallet discovery...");
			window.dispatchEvent(new Event("eip6963:requestProvider"));
			
			// Log available wallets for debugging
			setTimeout(() => {
				console.log("Checking wallet manager...");
				if (window.walletManager) {
					console.log("Available wallets:", window.walletManager.wallets.size);
					for (const [uuid, wallet] of window.walletManager.wallets) {
						console.log("  - " + uuid + ": " + wallet.info.name);
					}
				} else {
					console.log("No walletManager found");
				}
				
				// Check wallet select element (new data-testid approach)
				const walletSelect = document.querySelector('[data-testid="wallet-selector-dropdown"]');
				if (walletSelect) {
					console.log("Wallet selector dropdown found");
					// Check for wallet options
					const walletOptions = document.querySelectorAll('[data-testid^="wallet-selector-option-"]');
					console.log("Found", walletOptions.length, "wallet options");
					walletOptions.forEach((option, index) => {
						console.log("  Option", index + ":", option.textContent);
					});
				} else {
					console.log("No wallet-selector-dropdown element found");
				}
			}, 500);
		}, 100);
	`, s.walletProviderScript, privateKeyHex)

	err = chromedp.Run(s.ctx, chromedp.Evaluate(script, nil))
	if err != nil {
		return fmt.Errorf("failed to inject wallet provider: %w", err)
	}

	// Retrieve debug logs to see what happened
	var debugLogs []map[string]interface{}
	err = chromedp.Run(s.ctx, chromedp.Evaluate(`window.debugLogs || []`, &debugLogs))
	if err == nil && len(debugLogs) > 0 {
		s.t.Logf("Browser console logs:")
		for _, log := range debugLogs {
			if logType, ok := log["type"].(string); ok {
				if message, ok := log["message"].(string); ok {
					s.t.Logf("  [%s] %s", logType, message)
				}
			}
		}
	}

	return nil
}

// getOrCreateTestChain gets or creates a test chain for e2e testing
func (s *ChromedpTestSetup) getOrCreateTestChain() (*models.Chain, error) {
	// Try to get existing active chain
	activeChain, err := s.TestSetup.ChainService.GetActiveChain()
	if err == nil && activeChain != nil {
		return activeChain, nil
	}

	// Create a test chain
	testChain := &models.Chain{
		ChainType: "ethereum",
		RPC:       "http://localhost:8545",
		NetworkID: "31337",
		Name:      "Anvil Testnet",
		IsActive:  true,
	}

	err = s.TestSetup.ChainService.CreateChain(testChain)
	if err != nil {
		return nil, fmt.Errorf("failed to create test chain: %w", err)
	}

	return testChain, nil
}

// CreateUniswapDeploymentSession creates a Uniswap deployment session for testing
func (s *ChromedpTestSetup) CreateUniswapDeploymentSession() (string, error) {
	// Get or create test chain
	chain, err := s.getOrCreateTestChain()
	if err != nil {
		return "", fmt.Errorf("failed to get test chain: %w", err)
	}

	// First create the Uniswap deployment record
	deployment := &models.UniswapDeployment{
		Version: "v2",
		ChainID: chain.ID,
		Status:  models.TransactionStatusPending,
	}

	err = s.TestSetup.DBService.GetDB().Create(deployment).Error
	if err != nil {
		return "", fmt.Errorf("failed to create uniswap deployment: %w", err)
	}

	// Create session data that matches the expected format
	sessionData := map[string]interface{}{
		"uniswap_deployment_id": deployment.ID,
		"version":               "v2",
		"chain_id":              "31337",
		"metadata": map[string]interface{}{
			"version":   "v2",
			"chain":     "Ethereum Testnet",
			"initiator": "E2E Test",
		},
	}

	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Create transaction session in database
	sessionID, err := s.TestSetup.TxService.CreateTransactionSessionLegacy(
		"deploy_uniswap",
		models.TransactionChainTypeEthereum,
		"31337",
		string(sessionDataJSON),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction session: %w", err)
	}

	return sessionID, nil
}

// CreateOpenZeppelinTemplate creates an OpenZeppelin ERC20 template for testing
func (s *ChromedpTestSetup) CreateOpenZeppelinTemplate() (uint, error) {
	template := &models.Template{
		Name:        "Test OpenZeppelin ERC20",
		Description: "OpenZeppelin ERC20 token for testing",
		ChainType:   "ethereum",
		TemplateCode: `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract TestToken is ERC20 {
    constructor(string memory name, string memory symbol) ERC20(name, symbol) {
        _mint(msg.sender, 1000000 * 10 ** decimals());
    }
}`,
	}

	err := s.TestSetup.TemplateService.CreateTemplate(template)
	if err != nil {
		return 0, fmt.Errorf("failed to create OpenZeppelin template: %w", err)
	}

	return template.ID, nil
}

// CreateCustomTemplate creates a custom ERC20 template for testing
func (s *ChromedpTestSetup) CreateCustomTemplate() (uint, error) {
	template := &models.Template{
		Name:        "Test Custom ERC20",
		Description: "Custom ERC20 token for testing",
		ChainType:   "ethereum",
		TemplateCode: `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract CustomToken {
    string public name;
    string public symbol;
    uint8 public decimals = 18;
    uint256 public totalSupply;
    
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    
    constructor(string memory _name, string memory _symbol) {
        name = _name;
        symbol = _symbol;
        totalSupply = 1000000 * 10 ** decimals;
        balanceOf[msg.sender] = totalSupply;
    }
    
    function transfer(address _to, uint256 _value) public returns (bool) {
        require(balanceOf[msg.sender] >= _value, "Insufficient balance");
        balanceOf[msg.sender] -= _value;
        balanceOf[_to] += _value;
        emit Transfer(msg.sender, _to, _value);
        return true;
    }
    
    function approve(address _spender, uint256 _value) public returns (bool) {
        allowance[msg.sender][_spender] = _value;
        emit Approval(msg.sender, _spender, _value);
        return true;
    }
    
    function transferFrom(address _from, address _to, uint256 _value) public returns (bool) {
        require(balanceOf[_from] >= _value, "Insufficient balance");
        require(allowance[_from][msg.sender] >= _value, "Insufficient allowance");
        balanceOf[_from] -= _value;
        balanceOf[_to] += _value;
        allowance[_from][msg.sender] -= _value;
        emit Transfer(_from, _to, _value);
        return true;
    }
}`,
	}

	err := s.TestSetup.TemplateService.CreateTemplate(template)
	if err != nil {
		return 0, fmt.Errorf("failed to create custom template: %w", err)
	}

	return template.ID, nil
}

// CreateTokenDeploymentSession creates a token deployment session for testing using the proper TransactionService
func (s *ChromedpTestSetup) CreateTokenDeploymentSession(templateID uint) (string, error) {
	// Get the template to include in session data
	template, err := s.TestSetup.TemplateService.GetTemplateByID(templateID)
	if err != nil {
		return "", fmt.Errorf("failed to get template: %w", err)
	}

	// Get or create test chain
	chain, err := s.getOrCreateTestChain()
	if err != nil {
		return "", fmt.Errorf("failed to get test chain: %w", err)
	}

	// First create the deployment record
	deployment := &models.Deployment{
		TemplateID:  templateID,
		ChainID:     chain.ID,
		TokenName:   "Test Token",
		TokenSymbol: "TEST",
		Status:      "pending",
	}

	err = s.TestSetup.DeploymentService.CreateDeployment(deployment)
	if err != nil {
		return "", fmt.Errorf("failed to create deployment record: %w", err)
	}

	// Create a simple transaction deployment with basic contract bytecode for testing
	// We'll use a minimal ERC20-like bytecode that deploys successfully
	testBytecode := "0x608060405234801561001057600080fd5b50336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555061003e8061005d6000396000f3fe6080604052600080fdfea26469706673582212205c6b0e6b8f8e1a4c5e8e9f4d8f3b6c4e8f9f1c8e5f4c8e6f4c8e6f4c8e6f4c8e64736f6c63430008130033"

	// Create transaction session using the proper service (like launch.go does)
	sessionID, err := s.TestSetup.TxService.CreateTransactionSession(services.CreateTransactionSessionRequest{
		Metadata: []models.TransactionMetadata{
			{Key: "deployment_id", Value: fmt.Sprintf("%d", deployment.ID)},
			{Key: "template_id", Value: fmt.Sprintf("%d", templateID)},
			{Key: "template_name", Value: template.Name},
			{Key: "chain", Value: "Ethereum Testnet"},
			{Key: "initiator", Value: "E2E Test"},
		},
		TransactionDeployments: []models.TransactionDeployment{
			{
				Title:       fmt.Sprintf("Deploy %s", template.Name),
				Description: fmt.Sprintf("Deploy %s token contract", template.Name),
				Data:        testBytecode,
				Value:       "0",
				Receiver:    "0x0000000000000000000000000000000000000000",
				Status:      models.TransactionStatusPending,
			},
		},
		ChainType: models.TransactionChainTypeEthereum,
		ChainID:   chain.ID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create transaction session: %w", err)
	}

	return sessionID, nil
}

// GetBaseURL returns the base URL for the test server
func (s *ChromedpTestSetup) GetBaseURL() string {
	return fmt.Sprintf("http://localhost:%d", s.TestSetup.ServerPort)
}

// TakeScreenshotOnFailure takes a screenshot if the test fails
func (s *ChromedpTestSetup) TakeScreenshotOnFailure(t *testing.T, testName string) {
	if t.Failed() {
		filename := fmt.Sprintf("screenshot_%s_%d.png", testName, s.TestSetup.ServerPort)
		var buf []byte
		if err := chromedp.Run(s.ctx, chromedp.FullScreenshot(&buf, 90)); err == nil {
			os.WriteFile(filename, buf, 0644)
			t.Logf("Screenshot saved to: %s", filename)
		}
	}
}

// VerifyContractDeployment verifies that a contract was deployed successfully
func (s *ChromedpTestSetup) VerifyContractDeployment(contractAddress string) error {
	ctx := context.Background()

	address := common.HexToAddress(contractAddress)
	code, err := s.TestSetup.EthClient.CodeAt(ctx, address, nil)
	if err != nil {
		return fmt.Errorf("failed to get contract code: %w", err)
	}

	if len(code) == 0 {
		return fmt.Errorf("no contract code found at address %s", contractAddress)
	}

	return nil
}

// WaitForTransactionConfirmation waits for a transaction to be models.TransactionStatusConfirmed
func (s *ChromedpTestSetup) WaitForTransactionConfirmation(txHash string) error {
	hash := common.HexToHash(txHash)
	receipt, err := s.TestSetup.WaitForTransaction(hash, 60) // 60 second timeout
	if err != nil {
		return fmt.Errorf("transaction confirmation failed: %w", err)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("transaction failed with status %d", receipt.Status)
	}

	return nil
}

// VerifyEthereumConnection verifies connection to Ethereum testnet
func (s *TestSetup) VerifyEthereumConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.EthClient.NetworkID(ctx)
	return err
}

// GetPrimaryTestAccount returns the primary test account
func (s *ChromedpTestSetup) GetPrimaryTestAccount() *TestAccount {
	privateKey, err := crypto.HexToECDSA(TESTING_PK_1[2:]) // Remove 0x prefix
	require.NoError(s.TestSetup.t, err)

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	require.True(s.TestSetup.t, ok)

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &TestAccount{
		PrivateKey: privateKey,
		Address:    address,
	}
}

// TestAccount represents a test account
type TestAccount struct {
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
}

// WaitForTransaction waits for transaction confirmation
func (s *TestSetup) WaitForTransaction(txHash common.Hash, timeoutSeconds int) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for transaction %s", txHash.Hex())
		default:
			receipt, err := s.EthClient.TransactionReceipt(ctx, txHash)
			if err == nil {
				return receipt, nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// Cleanup properly shuts down the TestSetup
func (s *TestSetup) Cleanup() {
	if s.EthClient != nil {
		s.EthClient.Close()
	}
	if s.DBService != nil {
		s.DBService.Close()
	}
}

// Cleanup properly shuts down all test infrastructure
func (s *ChromedpTestSetup) Cleanup() {
	// Print console errors before cleanup
	if s.ctx != nil {
		var logs []map[string]interface{}
		err := chromedp.Run(s.ctx, chromedp.Evaluate(`
			(() => {
				const logs = [];
				if (window.testLogs) {
					logs.push(...window.testLogs);
				}
				// Get console errors from the console API if available
				if (console._originalError) {
					logs.push({level: 'error', message: 'Console errors were captured but details unavailable'});
				}
				return logs;
			})()
		`, &logs))

		if err == nil && len(logs) > 0 {
			s.t.Logf("Chrome console errors/logs:")
			for _, log := range logs {
				if level, ok := log["level"].(string); ok {
					if message, ok := log["message"].(string); ok {
						s.t.Logf("  [%s] %s", level, message)
					}
				}
			}
		}

		// Also try to get browser console logs via Chrome DevTools
		var consoleEntries []map[string]interface{}
		chromedp.Run(s.ctx, chromedp.Evaluate(`
			(() => {
				const entries = [];
				// Try to access any stored console messages
				if (window.consoleMessages) {
					return window.consoleMessages;
				}
				return [];
			})()
		`, &consoleEntries))

		if len(consoleEntries) > 0 {
			s.t.Logf("Additional console entries:")
			for _, entry := range consoleEntries {
				s.t.Logf("  %+v", entry)
			}
		}
	}

	// Shutdown API server
	if s.apiServer != nil {
		s.apiServer.Shutdown()
	}

	if s.cancel != nil {
		s.cancel()
	}
	s.TestSetup.Cleanup()
}
