// EIP-6963 Compliant Test Wallet Provider
// This script implements a test wallet provider that emits EIP-6963 events
// and provides Ethereum provider functionality for E2E testing.

class TestWalletProvider {
  constructor(privateKey, chainId = "0x7a69") {
    console.log("TestWalletProvider constructor called with privateKey:", privateKey.slice(0, 10) + "...", "chainId:", chainId);
    this.privateKey = privateKey;
    this.chainId = chainId;
    this.accounts = [];
    this.isConnected = false;
    this.eventTarget = new EventTarget();
    
    // Bind methods to maintain context
    this.request = this.request.bind(this);
    this.on = this.on.bind(this);
    this.removeListener = this.removeListener.bind(this);
    
    // Store reference for cleanup
    window._testWalletProvider = this;
    console.log("TestWalletProvider created and stored in window._testWalletProvider");
  }

  // Derive account address from private key
  async deriveAddress() {
    if (this.accounts.length === 0) {
      // Use exposed Go function to derive address from private key
      if (window.goSignTransaction) {
        const result = await window.goSignTransaction(JSON.stringify({
          action: "derive_address",
          privateKey: this.privateKey
        }));
        const parsed = JSON.parse(result);
        if (parsed.success) {
          this.accounts = [parsed.address];
        } else {
          throw new Error(`Failed to derive address: ${parsed.error}`);
        }
      } else {
        throw new Error("Go signing function not available");
      }
    }
    return this.accounts[0];
  }

  // EIP-1193 request method
  async request({ method, params = [] }) {
    console.log(`TestWallet request: ${method}`, params);

    switch (method) {
      case "eth_requestAccounts":
        await this.deriveAddress();
        this.isConnected = true;
        this.emit("accountsChanged", this.accounts);
        return this.accounts;

      case "eth_accounts":
        return this.isConnected ? this.accounts : [];

      case "eth_chainId":
        return this.chainId;

      case "net_version":
        return parseInt(this.chainId, 16).toString();

      case "eth_sendTransaction":
        return await this.signAndSendTransaction(params[0]);

      case "personal_sign":
        return await this.personalSign(params[0], params[1]);

      case "wallet_switchEthereumChain":
        const targetChainId = params[0].chainId;
        if (targetChainId !== this.chainId) {
          this.chainId = targetChainId;
          this.emit("chainChanged", this.chainId);
        }
        return null;

      case "wallet_addEthereumChain":
        // For testing, just accept any chain
        return null;

      case "eth_getBalance":
        // Mock balance - in real tests this would query the testnet
        return "0x1bc16d674ec80000"; // 2 ETH

      case "eth_getTransactionReceipt":
        return await this.getTransactionReceipt(params[0]);

      default:
        throw new Error(`Unsupported method: ${method}`);
    }
  }

  // Sign and send transaction using Go backend
  async signAndSendTransaction(transaction) {
    if (!this.isConnected) {
      throw new Error("Wallet not connected");
    }

    try {
      const result = await window.goSignTransaction(JSON.stringify({
        action: "sign_transaction",
        privateKey: this.privateKey,
        transaction: transaction
      }));

      const parsed = JSON.parse(result);
      if (parsed.success) {
        console.log(`Transaction signed and sent: ${parsed.txHash}`);
        return parsed.txHash;
      } else {
        throw new Error(`Transaction failed: ${parsed.error}`);
      }
    } catch (error) {
      console.error("Transaction signing failed:", error);
      throw error;
    }
  }

  // Personal sign implementation
  async personalSign(message, address) {
    if (!this.isConnected) {
      throw new Error("Wallet not connected");
    }

    try {
      const result = await window.goSignTransaction(JSON.stringify({
        action: "personal_sign",
        privateKey: this.privateKey,
        message: message,
        address: address
      }));

      const parsed = JSON.parse(result);
      if (parsed.success) {
        return parsed.signature;
      } else {
        throw new Error(`Signing failed: ${parsed.error}`);
      }
    } catch (error) {
      console.error("Personal sign failed:", error);
      throw error;
    }
  }

  // Get transaction receipt from testnet via direct RPC call
  async getTransactionReceipt(txHash) {
    try {
      const response = await fetch("http://localhost:8545", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          jsonrpc: "2.0",
          method: "eth_getTransactionReceipt",
          params: [txHash],
          id: 1
        })
      });

      const result = await response.json();
      
      if (result.error) {
        throw new Error(`RPC error: ${result.error.message}`);
      }
      
      // Return null if transaction is not yet mined (standard behavior)
      return result.result;
    } catch (error) {
      console.error("Get transaction receipt failed:", error);
      throw error;
    }
  }

  // Event handling
  on(event, callback) {
    this.eventTarget.addEventListener(event, callback);
  }

  removeListener(event, callback) {
    this.eventTarget.removeEventListener(event, callback);
  }

  emit(event, data) {
    this.eventTarget.dispatchEvent(new CustomEvent(event, { detail: data }));
  }

  // Check if provider is connected
  isConnected() {
    return this.isConnected;
  }
}

// EIP-6963 Provider Discovery Implementation
function announceProvider(privateKey, chainId) {
  console.log("announceProvider called with privateKey:", privateKey.slice(0, 10) + "...", "chainId:", chainId);
  const provider = new TestWalletProvider(privateKey, chainId);
  console.log("TestWalletProvider created:", provider);
  
  const info = {
    uuid: "test-wallet-e2e",
    name: "Test Wallet E2E",
    icon: "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTEyIDJMMTMuMDkgOC4yNkwyMCA5TDEzLjA5IDE1Ljc0TDEyIDIyTDEwLjkxIDE1Ljc0TDQgOUwxMC45MSA4LjI2TDEyIDJaIiBmaWxsPSIjMDA3OEZGIi8+Cjwvc3ZnPgo=",
    rdns: "io.test.wallet.e2e"
  };

  // Store the provider globally for access
  window.ethereum = provider;
  console.log("window.ethereum set to:", window.ethereum);
  
  // Announce the provider
  const announceEvent = new CustomEvent("eip6963:announceProvider", {
    detail: { info, provider }
  });
  
  console.log("Dispatching EIP-6963 announce event:", announceEvent);
  window.dispatchEvent(announceEvent);
  console.log("Test wallet provider announced via EIP-6963");
  
  return provider;
}

// Listen for provider requests
window.addEventListener("eip6963:requestProvider", () => {
  console.log("Received EIP-6963 provider request, checking for _testWalletProvider:", !!window._testWalletProvider);
  // Re-announce if we have a provider
  if (window._testWalletProvider) {
    console.log("Re-announcing existing test wallet provider");
    const info = {
      uuid: "test-wallet-e2e",
      name: "Test Wallet E2E", 
      icon: "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTEyIDJMMTMuMDkgOC4yNkwyMCA5TDEzLjA5IDE1Ljc0TDEyIDIyTDEwLjkxIDE1Ljc0TDQgOUwxMC45MSA4LjI2TDEyIDJaIiBmaWxsPSIjMDA3OEZGIi8+Cjwvc3ZnPgo=",
      rdns: "io.test.wallet.e2e"
    };
    
    const announceEvent = new CustomEvent("eip6963:announceProvider", {
      detail: { info, provider: window._testWalletProvider }
    });
    
    console.log("Re-dispatching EIP-6963 announce event");
    window.dispatchEvent(announceEvent);
  } else {
    console.log("No _testWalletProvider found to re-announce");
  }
});

// Initialize provider function (called by Go test)
window.initTestWallet = function(privateKey, chainId = "0x7a69") {
  console.log("initTestWallet called with privateKey:", privateKey.slice(0, 10) + "...", "chainId:", chainId);
  console.log("About to call announceProvider...");
  const provider = announceProvider(privateKey, chainId);
  console.log("announceProvider returned:", provider);
  console.log("window._testWalletProvider is now:", window._testWalletProvider);
  return provider;
};

// Cleanup function
window.cleanupTestWallet = function() {
  if (window._testWalletProvider) {
    delete window._testWalletProvider;
    delete window.ethereum;
  }
};

console.log("Test wallet provider script loaded");