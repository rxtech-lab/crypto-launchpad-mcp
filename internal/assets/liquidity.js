// Liquidity Management Scripts for Uniswap Operations
// This file handles all liquidity-related operations: create pool, add/remove liquidity, and swaps

class LiquidityManager {
  constructor() {
    this.sessionData = null;
    this.walletManager = null;
    this.isInitialized = false;
  }

  async init() {
    if (this.isInitialized) return;

    // Wait for wallet manager to be available
    await this.waitForWalletManager();

    // Initialize wallet manager
    this.walletManager = window.walletManager;
    if (!this.walletManager) {
      throw new Error("WalletManager not found");
    }

    // Load session data
    await this.loadSessionData();

    // Display transaction details
    this.displayTransactionDetails();

    // Setup event listeners
    this.setupEventListeners();

    this.isInitialized = true;
  }

  async waitForWalletManager() {
    let attempts = 0;
    while (!window.walletManager && attempts < 50) {
      await new Promise((resolve) => setTimeout(resolve, 100));
      attempts++;
    }
    if (!window.walletManager) {
      throw new Error("WalletManager not initialized after 5 seconds");
    }
  }

  async loadSessionData() {
    const sessionElement = document.getElementById("session-data");
    if (!sessionElement) {
      throw new Error("Session data element not found");
    }

    const sessionId = sessionElement.dataset.sessionId;
    const apiUrl = sessionElement.dataset.apiUrl;
    const embeddedData = sessionElement.dataset.transactionData;

    // Try embedded data first
    if (embeddedData) {
      try {
        this.sessionData = JSON.parse(embeddedData);
        console.log("Using embedded transaction data");
        return;
      } catch (error) {
        console.warn("Failed to parse embedded data:", error);
      }
    }

    // Fallback to API call
    if (apiUrl) {
      try {
        const response = await fetch(apiUrl);
        if (!response.ok) {
          throw new Error(`Failed to fetch session data: ${response.status}`);
        }
        this.sessionData = await response.json();
        console.log("Loaded session data from API");
      } catch (error) {
        console.error("Failed to load session data:", error);
        throw error;
      }
    }
  }

  displayTransactionDetails() {
    const contentElement = document.getElementById("content");
    if (!contentElement || !this.sessionData) return;

    const { transaction_data, session_type } = this.sessionData;
    let detailsHTML = "";

    try {
      switch (session_type) {
        case "create_pool":
          detailsHTML = this.generateCreatePoolDetails(transaction_data || {});
          break;
        case "add_liquidity":
          detailsHTML = this.generateAddLiquidityDetails(
            transaction_data || {}
          );
          break;
        case "remove_liquidity":
          detailsHTML = this.generateRemoveLiquidityDetails(
            transaction_data || {}
          );
          break;
        case "swap":
          detailsHTML = this.generateSwapDetails(transaction_data || {});
          break;
        default:
          detailsHTML =
            '<p class="text-red-600">Unknown liquidity operation type</p>';
      }
    } catch (error) {
      console.error("Error generating transaction details:", error);
      detailsHTML = `<p class="text-red-600">Error loading transaction details: ${error.message}</p>`;
    }

    contentElement.innerHTML = `
      <div class="max-w-2xl mx-auto space-y-6">
        <!-- Transaction Details Card -->
        <div class="bg-white border border-gray-200 rounded-2xl p-6 shadow-sm">
          <div class="flex items-center mb-4">
            <div class="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center mr-3">
              <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
            </div>
            <div>
              <h3 class="font-semibold text-gray-900">${this.getOperationTitle(
                session_type
              )}</h3>
              <p class="text-sm text-gray-600">Review and confirm transaction</p>
            </div>
          </div>
          ${detailsHTML}
        </div>

        <!-- Wallet Connection Card -->
        <div class="bg-white border border-gray-200 rounded-2xl p-6 shadow-sm">
          <h3 class="font-semibold text-gray-900 mb-4">Connect Wallet</h3>
          
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-2">Select Wallet</label>
            <select id="wallet-select" class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500">
              <option value="">Choose a wallet...</option>
            </select>
          </div>

          <button id="connect-wallet" class="w-full bg-gradient-to-r from-blue-500 to-blue-600 text-white font-medium py-3 px-4 rounded-xl hover:from-blue-600 hover:to-blue-700 transition-all duration-200 shadow-sm">
            Connect Wallet
          </button>

          <div id="wallet-info" class="mt-4"></div>
        </div>

        <!-- Sign Transaction Button (initially hidden) -->
        <button id="${this.getButtonId(
          session_type
        )}" class="hidden w-full bg-gradient-to-r from-green-500 to-green-600 text-white font-medium py-3 px-4 rounded-xl hover:from-green-600 hover:to-green-700 transition-all duration-200 shadow-sm disabled:opacity-50 disabled:cursor-not-allowed" disabled>
          ${this.getButtonText(session_type)}
        </button>

        <!-- Status Messages -->
        <div id="status-message" class="hidden"></div>
      </div>
    `;
  }

  getOperationTitle(sessionType) {
    switch (sessionType) {
      case "create_pool":
        return "Create Liquidity Pool";
      case "add_liquidity":
        return "Add Liquidity";
      case "remove_liquidity":
        return "Remove Liquidity";
      case "swap":
        return "Swap Tokens";
      default:
        return "Liquidity Operation";
    }
  }

  getButtonId(sessionType) {
    switch (sessionType) {
      case "create_pool":
        return "create-pool-btn";
      case "add_liquidity":
        return "add-liquidity-btn";
      case "remove_liquidity":
        return "remove-liquidity-btn";
      case "swap":
        return "swap-btn";
      default:
        return "sign-button";
    }
  }

  getButtonText(sessionType) {
    switch (sessionType) {
      case "create_pool":
        return "Create Pool";
      case "add_liquidity":
        return "Add Liquidity";
      case "remove_liquidity":
        return "Remove Liquidity";
      case "swap":
        return "Swap Tokens";
      default:
        return "Sign Transaction";
    }
  }

  generateCreatePoolDetails(data) {
    // The data might just have pool_id from the test, handle gracefully
    if (!data || Object.keys(data).length === 0) {
      return `
        <div class="space-y-3">
          <p class="text-gray-600">Pool details will be loaded...</p>
        </div>
      `;
    }

    return `
      <div class="space-y-3">
        ${
          data.token_address
            ? `
        <div class="py-2">
          <span class="text-gray-600 block mb-1">Token Address</span>
          <code class="text-xs bg-gray-100 px-2 py-1 rounded-md text-gray-800 break-all">${data.token_address}</code>
        </div>
        `
            : ""
        }
        ${
          data.initial_token0
            ? `
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Initial Token Amount</span>
          <span class="font-medium text-gray-900">${data.initial_token0}</span>
        </div>
        `
            : ""
        }
        ${
          data.initial_token1
            ? `
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Initial ETH Amount</span>
          <span class="font-medium text-gray-900">${data.initial_token1} ETH</span>
        </div>
        `
            : ""
        }
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Protocol</span>
          <span class="font-medium text-purple-600">${
            data.uniswap_version || "V2"
          }</span>
        </div>
        ${
          data.pool_id
            ? `
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Pool ID</span>
          <span class="text-sm text-gray-700">#${data.pool_id}</span>
        </div>
        `
            : ""
        }
      </div>
    `;
  }

  generateAddLiquidityDetails(data) {
    // Handle both pool_address and token_address (from test data)
    const address = data.pool_address || data.token_address || "N/A";
    const tokenAmount = data.token0_amount || data.token_amount || "0";
    const ethAmount = data.token1_amount || data.eth_amount || "0";

    return `
      <div class="space-y-3">
        <div class="py-2">
          <span class="text-gray-600 block mb-1">Token/Pool Address</span>
          <code class="text-xs bg-gray-100 px-2 py-1 rounded-md text-gray-800 break-all">${address}</code>
        </div>
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Token Amount</span>
          <span class="font-medium text-gray-900">${tokenAmount}</span>
        </div>
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">ETH Amount</span>
          <span class="font-medium text-gray-900">${ethAmount} ETH</span>
        </div>
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Slippage Tolerance</span>
          <span class="text-sm text-gray-700">${data.slippage || "2"}%</span>
        </div>
        ${
          data.min_token_amount
            ? `
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Min Token Amount</span>
          <span class="text-sm text-gray-700">${data.min_token_amount}</span>
        </div>
        `
            : ""
        }
        ${
          data.min_eth_amount
            ? `
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Min ETH Amount</span>
          <span class="text-sm text-gray-700">${data.min_eth_amount}</span>
        </div>
        `
            : ""
        }
      </div>
    `;
  }

  generateRemoveLiquidityDetails(data) {
    const address = data.pool_address || data.token_address || "N/A";
    const liquidityAmount =
      data.liquidity_amount || data.position_amount || "0";

    return `
      <div class="space-y-3">
        <div class="py-2">
          <span class="text-gray-600 block mb-1">Pool/Token Address</span>
          <code class="text-xs bg-gray-100 px-2 py-1 rounded-md text-gray-800 break-all">${address}</code>
        </div>
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Liquidity Amount</span>
          <span class="font-medium text-red-600">${liquidityAmount}</span>
        </div>
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Min Token Amount</span>
          <span class="text-sm text-gray-700">${
            data.min_token0 || data.min_token_amount || "0"
          }</span>
        </div>
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Min ETH Amount</span>
          <span class="text-sm text-gray-700">${
            data.min_token1 || data.min_eth_amount || "0"
          } ETH</span>
        </div>
      </div>
    `;
  }

  generateSwapDetails(data) {
    const fromDisplay = data.from_token === "0x0" ? "ETH" : data.from_token;
    const toDisplay = data.to_token === "0x0" ? "ETH" : data.to_token;

    return `
      <div class="space-y-3">
        <div class="bg-blue-50 rounded-xl p-4">
          <div class="flex items-center justify-between">
            <div class="flex flex-col">
              <span class="text-sm text-gray-600 mb-1">From</span>
              <span class="font-mono text-sm bg-white px-3 py-1 rounded-lg">${
                fromDisplay === "ETH" ? "ETH" : fromDisplay.slice(0, 8) + "..."
              }</span>
              <span class="font-medium text-lg mt-1">${
                data.from_amount || "0"
              }</span>
            </div>
            <div class="px-4">
              <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3"></path>
              </svg>
            </div>
            <div class="flex flex-col">
              <span class="text-sm text-gray-600 mb-1">To</span>
              <span class="font-mono text-sm bg-white px-3 py-1 rounded-lg">${
                toDisplay === "ETH" ? "ETH" : toDisplay.slice(0, 8) + "..."
              }</span>
              <span class="font-medium text-lg mt-1">${
                data.to_amount || "0"
              }</span>
            </div>
          </div>
        </div>
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Slippage Tolerance</span>
          <span class="text-sm text-gray-700">${
            data.slippage_tolerance || "0.5"
          }%</span>
        </div>
        <div class="flex justify-between items-center py-2 border-t border-gray-100">
          <span class="text-gray-600">Price Impact</span>
          <span class="text-sm ${
            data.price_impact > 5 ? "text-red-600" : "text-gray-700"
          }">${data.price_impact || "< 0.01"}%</span>
        </div>
      </div>
    `;
  }

  setupEventListeners() {
    // Connect wallet button
    const connectBtn = document.getElementById("connect-wallet");
    if (connectBtn) {
      connectBtn.addEventListener("click", () => this.handleConnectWallet());
    }

    // Sign button - use dynamic button ID based on session type
    const buttonId = this.getButtonId(this.sessionData.session_type);
    const signBtn = document.getElementById(buttonId);
    if (signBtn) {
      signBtn.addEventListener("click", () => this.handleSignTransaction());
    }

    // Listen for wallet updates
    window.addEventListener("walletsUpdated", () => {
      this.walletManager.updateWalletList();
    });
  }

  async handleConnectWallet() {
    try {
      const walletSelect = document.getElementById("wallet-select");
      const selectedWallet = walletSelect.value;

      if (!selectedWallet) {
        this.showStatus("Please select a wallet", "error");
        return;
      }

      const statusDiv = document.getElementById("wallet-info");
      statusDiv.innerHTML = '<p class="text-blue-600">Connecting...</p>';

      await this.walletManager.connectWallet(selectedWallet);

      const account = this.walletManager.getAccount();
      statusDiv.innerHTML = `
        <div class="bg-green-50 border border-green-200 rounded-lg p-3">
          <p class="text-green-800 text-sm">Connected: ${account.slice(
            0,
            6
          )}...${account.slice(-4)}</p>
        </div>
      `;

      // Show and enable sign button - use dynamic button ID
      const buttonId = this.getButtonId(this.sessionData.session_type);
      const signBtn = document.getElementById(buttonId);
      if (signBtn) {
        signBtn.classList.remove("hidden");
        signBtn.disabled = false;
      }
    } catch (error) {
      console.error("Failed to connect wallet:", error);
      this.showStatus("Failed to connect wallet: " + error.message, "error");
    }
  }

  async handleSignTransaction() {
    try {
      const buttonId = this.getButtonId(this.sessionData.session_type);
      const signBtn = document.getElementById(buttonId);
      signBtn.disabled = true;
      signBtn.textContent = "Processing...";

      // Get the transaction data based on session type
      const txData = await this.prepareTransaction();

      // Send transaction
      const txHash = await this.walletManager.signTransaction(txData);

      // Update UI for success
      await this.handleTransactionSuccess(txHash);
    } catch (error) {
      console.error("Transaction failed:", error);
      this.showStatus("Transaction failed: " + error.message, "error");

      const buttonId = this.getButtonId(this.sessionData.session_type);
      const signBtn = document.getElementById(buttonId);
      signBtn.disabled = false;
      signBtn.textContent = this.getButtonText(this.sessionData.session_type);
    }
  }

  async prepareTransaction() {
    const { session_type, transaction_data = {} } = this.sessionData;

    // This is a simplified transaction preparation
    // In production, this would use proper contract ABIs and encoding
    const baseTransaction = {
      from: this.walletManager.getAccount(),
      gas: "0x76c0", // 30000 gas
      gasPrice: "0x9184e72a000", // 10 gwei
    };

    switch (session_type) {
      case "create_pool":
        // Mock create pool transaction
        return {
          ...baseTransaction,
          to:
            transaction_data.factory_address ||
            "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f",
          data: "0x", // Would be encoded createPair call
          value: "0x0",
        };

      case "add_liquidity":
        // Mock add liquidity transaction
        return {
          ...baseTransaction,
          to:
            transaction_data.router_address ||
            "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D",
          data: "0x", // Would be encoded addLiquidityETH call
          value: transaction_data.eth_amount || "0x0",
        };

      case "remove_liquidity":
        // Mock remove liquidity transaction
        return {
          ...baseTransaction,
          to:
            transaction_data.router_address ||
            "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D",
          data: "0x", // Would be encoded removeLiquidityETH call
          value: "0x0",
        };

      case "swap":
        // Mock swap transaction
        return {
          ...baseTransaction,
          to:
            transaction_data.router_address ||
            "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D",
          data: "0x", // Would be encoded swap call
          value:
            transaction_data.from_token === "0x0"
              ? transaction_data.from_amount
              : "0x0",
        };

      default:
        throw new Error("Unknown transaction type");
    }
  }

  async handleTransactionSuccess(txHash) {
    // Update session status
    await this.updateSessionStatus(txHash);

    // Hide main content and show success state
    const content = document.getElementById("content");
    if (content) content.style.display = "none";

    const successState = document.getElementById("success-state");
    if (successState) {
      successState.classList.remove("hidden");

      // Update success message based on operation type
      const { session_type } = this.sessionData;
      const titleElement = successState.querySelector("h3");
      const messageElement = successState.querySelector("p");

      switch (session_type) {
        case "create_pool":
          titleElement.textContent = "Pool Created Successfully!";
          messageElement.textContent = "Your liquidity pool has been created.";

          // Show pair address if available
          const pairAddress = document.getElementById("pair-address");
          if (pairAddress) {
            pairAddress.textContent = "0x1234...5678"; // Mock address, would be from contract event
          }
          break;

        case "add_liquidity":
          titleElement.textContent = "Liquidity Added Successfully!";
          messageElement.textContent =
            "Your liquidity has been added to the pool.";
          break;

        case "remove_liquidity":
          titleElement.textContent = "Liquidity Removed Successfully!";
          messageElement.textContent =
            "Your liquidity has been removed from the pool.";
          break;

        case "swap":
          titleElement.textContent = "Swap Completed Successfully!";
          messageElement.textContent = "Your token swap has been executed.";
          break;
      }

      // Show transaction hash
      const txHashElement = document.getElementById("transaction-hash");
      if (txHashElement) {
        txHashElement.textContent = txHash;
      }
    }
  }

  async updateSessionStatus(txHash) {
    const sessionElement = document.getElementById("session-data");
    const sessionId = sessionElement.dataset.sessionId;
    const { session_type } = this.sessionData;

    // Determine the confirmation endpoint based on session type
    let endpoint = "";
    switch (session_type) {
      case "create_pool":
        endpoint = `/api/pool/create/${sessionId}/confirm`;
        break;
      case "add_liquidity":
        endpoint = `/api/liquidity/add/${sessionId}/confirm`;
        break;
      case "remove_liquidity":
        endpoint = `/api/liquidity/remove/${sessionId}/confirm`;
        break;
      case "swap":
        endpoint = `/api/swap/${sessionId}/confirm`;
        break;
    }

    try {
      const response = await fetch(endpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          transaction_hash: txHash,
          status: "confirmed",
          pair_address:
            session_type === "create_pool" ? "0x1234...5678" : undefined, // Mock for pool creation
        }),
      });

      if (!response.ok) {
        console.error("Failed to update session status");
      }
    } catch (error) {
      console.error("Failed to update session status:", error);
    }
  }

  showStatus(message, type = "info") {
    const statusDiv = document.getElementById("status-message");
    if (!statusDiv) return;

    statusDiv.classList.remove("hidden");
    statusDiv.className = `mt-4 p-4 rounded-lg ${
      type === "error"
        ? "bg-red-50 border border-red-200 text-red-800"
        : type === "success"
        ? "bg-green-50 border border-green-200 text-green-800"
        : "bg-blue-50 border border-blue-200 text-blue-800"
    }`;
    statusDiv.textContent = message;
  }
}

// Initialize liquidity manager when DOM is ready
document.addEventListener("DOMContentLoaded", async () => {
  const liquidityManager = new LiquidityManager();

  try {
    await liquidityManager.init();
    console.log("Liquidity manager initialized successfully");
  } catch (error) {
    console.error("Failed to initialize liquidity manager:", error);
    // Show error to user
    const contentElement = document.getElementById("content");
    if (contentElement) {
      contentElement.innerHTML = `
        <div class="bg-red-50 border border-red-200 rounded-lg p-4">
          <p class="text-red-800">Failed to load transaction details: ${error.message}</p>
        </div>
      `;
    }
  }
});
