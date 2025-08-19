// Create Pool specific functionality
class CreatePoolManager {
  constructor() {
    this.sessionData = null;
    this.walletManager = null;
    this.init();
  }

  async init() {
    console.log("CreatePoolManager initializing...");

    // Get session data from HTML
    const sessionElement = document.getElementById("session-data");
    if (!sessionElement) {
      console.error("Session data element not found");
      this.showError("Session data not found");
      return;
    }

    const sessionId = sessionElement.dataset.sessionId;
    const apiUrl = sessionElement.dataset.apiUrl;
    const embeddedData = sessionElement.dataset.transactionData;

    if (!sessionId) {
      console.error("Session ID not found");
      this.showError("Invalid session");
      return;
    }

    // Load session data
    await this.loadSessionData(sessionId, apiUrl, embeddedData);

    // Set up wallet manager
    this.setupWalletManager();
  }

  async loadSessionData(sessionId, apiUrl, embeddedData = null) {
    try {
      // Try embedded data first
      if (embeddedData) {
        console.log("Using embedded transaction data");
        this.sessionData = JSON.parse(embeddedData);
        this.displayTransactionDetails();
        return;
      }

      // Fallback to API call
      if (apiUrl) {
        const response = await fetch(apiUrl);
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${await response.text()}`);
        }
        const data = await response.json();
        this.sessionData = data.transaction_data || data;
        this.displayTransactionDetails();
      }
    } catch (error) {
      console.error("Error loading session data:", error);
      this.showError("Failed to load transaction details");
    }
  }

  setupWalletManager() {
    // Wait for wallet manager to be available
    if (window.walletManager) {
      this.walletManager = window.walletManager;
    } else {
      // Initialize a new wallet manager if needed
      this.walletManager = new WalletManager();
      window.walletManager = this.walletManager;
    }

    // Set up connect button handler
    const connectBtn = document.getElementById("connect-wallet");
    if (connectBtn) {
      connectBtn.addEventListener("click", async () => {
        const walletSelect = document.getElementById("wallet-select");
        const selectedWallet = walletSelect?.value;

        if (!selectedWallet) {
          alert("Please select a wallet");
          return;
        }

        try {
          await this.walletManager.connectWallet(selectedWallet);
          this.handleWalletConnected();
        } catch (error) {
          console.error("Failed to connect wallet:", error);
          alert("Failed to connect wallet: " + error.message);
        }
      });
    }
  }

  displayTransactionDetails() {
    if (!this.sessionData) return;

    const content = document.getElementById("content");
    if (!content) return;

    // Display pool creation details
    let html = `
            <div class="space-y-4">
                <div class="bg-gray-50 p-4 rounded">
                    <h3 class="font-semibold text-gray-700 mb-2">Pool Details</h3>
        `;

    if (this.sessionData.token_address) {
      html += `
                    <div class="text-sm space-y-1">
                        <div><span class="font-medium">Token Address:</span> 
                            <span id="token-address" class="font-mono text-xs break-all">${this.sessionData.token_address}</span>
                        </div>
            `;
    }

    if (this.sessionData.initial_token_amount) {
      html += `
                        <div><span class="font-medium">Initial Token Amount:</span> 
                            <span id="token-amount">${this.sessionData.initial_token_amount}</span>
                        </div>
            `;
    }

    if (this.sessionData.initial_eth_amount) {
      html += `
                        <div><span class="font-medium">Initial ETH Amount:</span> 
                            <span id="eth-amount">${this.sessionData.initial_eth_amount}</span>
                        </div>
            `;
    }

    html += `
                    </div>
                </div>

                <!-- Wallet Connection -->
                <div class="bg-blue-50 p-4 rounded">
                    <h3 class="font-semibold text-blue-700 mb-2">Connect Wallet</h3>
                    <div id="wallet-section">
                        <select id="wallet-select" class="w-full p-2 border rounded mb-2">
                            <option value="">Select a wallet...</option>
                        </select>
                        <button id="connect-wallet" class="w-full bg-blue-500 text-white py-2 px-4 rounded hover:bg-blue-600">
                            Connect Wallet
                        </button>
                    </div>
                    <div id="wallet-info" class="hidden mt-2 p-2 bg-white rounded">
                        <div class="text-sm">
                            <span class="font-medium">Connected:</span>
                            <span id="wallet-address" class="font-mono text-xs"></span>
                        </div>
                    </div>
                </div>

                <!-- Action Button -->
                <div id="action-section" class="hidden">
                    <button id="create-pool-btn" class="w-full bg-green-500 text-white py-3 px-4 rounded hover:bg-green-600 font-semibold">
                        Create Liquidity Pool
                    </button>
                </div>

                <!-- Status Messages -->
                <div id="status-section" class="hidden">
                    <div id="status-message" class="p-4 rounded"></div>
                </div>

                <!-- Success State -->
                <div id="success-state" class="hidden">
                    <div class="bg-green-50 p-4 rounded">
                        <h3 class="font-semibold text-green-700 mb-2">Pool Created Successfully!</h3>
                        <div class="text-sm space-y-1">
                            <div>
                                <span class="font-medium">Pair Address:</span>
                                <span id="pair-address" class="font-mono text-xs break-all"></span>
                            </div>
                            <div>
                                <span class="font-medium">Transaction Hash:</span>
                                <span id="transaction-hash" class="font-mono text-xs break-all"></span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;

    content.innerHTML = html;

    // Set up create pool button handler
    const createBtn = document.getElementById("create-pool-btn");
    if (createBtn) {
      createBtn.addEventListener("click", () => this.handleCreatePool());
    }
  }

  handleWalletConnected() {
    console.log("Wallet connected, showing action button");

    // Show wallet info
    const walletInfo = document.getElementById("wallet-info");
    if (walletInfo) {
      walletInfo.classList.remove("hidden");
      const walletAddress = document.getElementById("wallet-address");
      if (walletAddress && this.walletManager && this.walletManager.account) {
        walletAddress.textContent = this.walletManager.account;
      }
    }

    // Hide wallet selection
    const walletSection = document.getElementById("wallet-section");
    if (walletSection) {
      walletSection.style.display = "none";
    }

    // Show action button
    const actionSection = document.getElementById("action-section");
    if (actionSection) {
      actionSection.classList.remove("hidden");
    }
  }

  async handleCreatePool() {
    console.log("handleCreatePool called, walletManager:", this.walletManager);
    console.log("walletManager.provider:", this.walletManager?.provider);
    console.log("walletManager.account:", this.walletManager?.account);

    if (!this.walletManager || !this.walletManager.account) {
      this.showError("Please connect wallet first");
      return;
    }

    // Get provider from walletManager or window
    const provider =
      this.walletManager.provider ||
      window.ethereum ||
      window._testWalletProvider;

    const createBtn = document.getElementById("create-pool-btn");
    if (createBtn) {
      createBtn.disabled = true;
      createBtn.textContent = "Creating Pool...";
    }

    try {
      // Mock transaction for testing
      const mockTx = {
        to: "0x5FbDB2315678afecb367f032d93F642f64180aa3", // Mock factory address
        data: "0x12345678", // Mock transaction data
        value: "0x0",
        from: this.walletManager.account,
      };

      this.showStatus("Sending transaction...", "info");

      // Sign and send transaction using provider
      const txHash = await provider.request({
        method: "eth_sendTransaction",
        params: [mockTx],
      });

      if (txHash) {
        // Mock pair address for testing
        const mockPairAddress =
          "0x" +
          Array.from(crypto.getRandomValues(new Uint8Array(20)))
            .map((b) => b.toString(16).padStart(2, "0"))
            .join("");

        this.showSuccess(txHash, mockPairAddress);
        await this.confirmTransaction(txHash, mockPairAddress);
      }
    } catch (error) {
      console.error("Error creating pool:", error);
      this.showError("Failed to create pool: " + error.message);
    } finally {
      if (createBtn) {
        createBtn.disabled = false;
        createBtn.textContent = "Create Liquidity Pool";
      }
    }
  }

  async confirmTransaction(txHash, pairAddress) {
    const sessionElement = document.getElementById("session-data");
    const sessionId = sessionElement?.dataset.sessionId;

    if (!sessionId) return;

    try {
      const response = await fetch(`/api/pool/create/${sessionId}/confirm`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          transaction_hash: txHash,
          status: "confirmed",
          pair_address: pairAddress,
        }),
      });

      if (!response.ok) {
        console.error("Failed to confirm transaction");
      }
    } catch (error) {
      console.error("Error confirming transaction:", error);
    }
  }

  showSuccess(txHash, pairAddress) {
    // Hide action section
    const actionSection = document.getElementById("action-section");
    if (actionSection) {
      actionSection.classList.add("hidden");
    }

    // Show success state
    const successState = document.getElementById("success-state");
    if (successState) {
      successState.classList.remove("hidden");

      const pairAddressEl = document.getElementById("pair-address");
      if (pairAddressEl) {
        pairAddressEl.textContent = pairAddress;
      }

      const txHashEl = document.getElementById("transaction-hash");
      if (txHashEl) {
        txHashEl.textContent = txHash;
      }
    }

    this.showStatus("Pool created successfully!", "success");
  }

  showStatus(message, type = "info") {
    const statusSection = document.getElementById("status-section");
    const statusMessage = document.getElementById("status-message");

    if (statusSection && statusMessage) {
      statusSection.classList.remove("hidden");
      statusMessage.textContent = message;
      statusMessage.className = `p-4 rounded ${
        type === "error"
          ? "bg-red-100 text-red-700"
          : type === "success"
          ? "bg-green-100 text-green-700"
          : "bg-blue-100 text-blue-700"
      }`;
    }
  }

  showError(message) {
    this.showStatus(message, "error");
  }
}

// Initialize when DOM is ready
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => {
    window.createPoolManager = new CreatePoolManager();
  });
} else {
  window.createPoolManager = new CreatePoolManager();
}
