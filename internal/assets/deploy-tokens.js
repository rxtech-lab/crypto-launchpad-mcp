// Token Deployment Management
class TokenDeploymentManager {
  constructor(walletManager) {
    this.walletManager = walletManager;
    this.sessionData = null;
    this.apiUrl = null;
  }

  async loadSessionData(sessionId, apiUrl, embeddedData = null) {
    this.apiUrl = apiUrl;

    // Check for embedded transaction data first
    if (embeddedData) {
      console.log("Using embedded transaction data");
      this.sessionData = {
        session_id: sessionId,
        session_type: "deploy",
        transaction_data: embeddedData,
        status: "pending",
      };
      this.displayTransactionDetails();
      return;
    }

    // Fallback to API call
    try {
      const response = await fetch(apiUrl);
      if (!response.ok) {
        throw new Error("Failed to load session data");
      }
      this.sessionData = await response.json();
      this.displayTransactionDetails();
    } catch (error) {
      console.error("Error loading session data:", error);
      this.displayError("Failed to load transaction details");
    }
  }

  displayTransactionDetails() {
    const contentElement = document.getElementById("content");
    if (!contentElement || !this.sessionData) return;

    const { transaction_data, session_type } = this.sessionData;

    if (session_type !== "deploy") {
      console.error("Invalid session type for token deployment:", session_type);
      this.displayError("Invalid session type");
      return;
    }

    const detailsHTML = this.generateDeploymentDetails(transaction_data);

    contentElement.innerHTML = `
            <div class="max-w-2xl mx-auto space-y-6 p-6">
                <!-- Header -->
                <div class="text-center mb-8">
                    <div class="w-16 h-16 mx-auto bg-blue-100 rounded-full flex items-center justify-center mb-4">
                        <svg class="w-8 h-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
                        </svg>
                    </div>
                    <h2 class="text-xl font-semibold text-gray-900 mb-2">Deploy Smart Contract</h2>
                    <p class="text-gray-600 text-sm mb-6">Review and confirm your contract deployment</p>
                </div>

                <!-- Transaction Details -->
                <div class="bg-gray-50 rounded-2xl p-6 mb-6">
                    <h3 class="text-lg font-semibold text-gray-900 mb-4">Transaction Details</h3>
                    ${detailsHTML}
                </div>

                <!-- Wallet Connection -->
                <div class="space-y-4">
                    <div>
                        <label for="wallet-select" class="block text-sm font-medium text-gray-700 mb-2">Select Wallet</label>
                        <select id="wallet-select" class="w-full px-4 py-3 bg-white border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent">
                            <option value="">Loading wallets...</option>
                        </select>
                    </div>
                    
                    <button id="connect-button" onclick="connectWallet()" class="w-full bg-blue-600 text-white py-4 px-6 rounded-xl font-medium hover:bg-blue-700 active:bg-blue-800 transition-all transform hover:scale-[1.02] active:scale-[0.98] shadow-lg">
                        Connect Wallet
                    </button>
                </div>

                <!-- Connection Status -->
                <div id="connection-status"></div>

                <!-- Transaction Status -->
                <div id="transaction-status" class="hidden">
                    <div id="status-message" class="text-center py-4"></div>
                </div>

                <!-- Sign Transaction Button -->
                <button id="sign-button" onclick="signTransaction()" style="display: none;" class="w-full bg-green-600 text-white py-4 px-6 rounded-xl font-medium hover:bg-green-700 active:bg-green-800 transition-all transform hover:scale-[1.02] active:scale-[0.98] shadow-lg">
                    Sign & Deploy Contract
                </button>
            </div>
        `;

    // Update wallet connection status
    this.walletManager.updateConnectionStatus();
  }

  generateDeploymentDetails(data) {
    return `
            <div class="space-y-4">
                <!-- Token Info Grid -->
                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div class="bg-gray-50 rounded-xl p-4">
                        <span class="text-gray-600 text-sm font-medium">Token Name</span>
                        <div class="font-semibold text-gray-900 text-lg mt-1">${data.token_name}</div>
                    </div>
                    <div class="bg-gray-50 rounded-xl p-4">
                        <span class="text-gray-600 text-sm font-medium">Symbol</span>
                        <div class="font-mono text-lg bg-white px-3 py-1 rounded-lg text-gray-900 mt-1 inline-block">${data.token_symbol}</div>
                    </div>
                </div>
                
                <!-- Template Info -->
                <div class="flex justify-between items-center py-3 border-t border-gray-200">
                    <span class="text-gray-600 font-medium">Contract Template</span>
                    <span class="text-sm text-gray-900 bg-gray-100 px-3 py-1 rounded-full">Smart Contract</span>
                </div>
            </div>
        `;
  }

  async prepareTransactionData() {
    const { transaction_data } = this.sessionData;

    // For contract deployment, we don't specify 'to' address and use the compiled bytecode as data
    const isContractDeployment =
      !transaction_data.contract_address && !transaction_data.token_address;

    // Fetch dynamic gas price and estimate gas limit
    const [gasPrice, estimatedGas] = await Promise.all([
      this.fetchGasPrice(),
      this.estimateGas(transaction_data, isContractDeployment),
    ]);

    let txData = {
      from: this.walletManager.getAccount(),
      value: transaction_data.eth_amount
        ? `0x${parseInt(transaction_data.eth_amount).toString(16)}`
        : "0x0",
      gas: estimatedGas,
      gasPrice: gasPrice,
      chainId: this.walletManager.getChainId(),
    };

    if (isContractDeployment && transaction_data.bytecode) {
      // Contract deployment - no 'to' address, use bytecode as data
      const bytecode =
        transaction_data.bytecode &&
        typeof transaction_data.bytecode === "string" &&
        transaction_data.bytecode.startsWith("0x")
          ? transaction_data.bytecode
          : "0x" + (transaction_data.bytecode || "");

      // If we have constructor parameters, we need to encode them and append to bytecode
      if (transaction_data.token_name && transaction_data.token_symbol) {
        // For now, use the bytecode directly - ABI encoding would be more complex
        txData.data = bytecode;
      } else {
        txData.data = bytecode;
      }
    } else {
      // Contract call or transfer - specify recipient and minimal data
      txData.to =
        transaction_data.contract_address || transaction_data.token_address;
      txData.data = "0x";
    }

    return txData;
  }

  async fetchGasPrice() {
    try {
      const gasPrice = await this.walletManager.selectedWallet.request({
        method: "eth_gasPrice",
        params: [],
      });

      // Add 10% buffer to gas price for faster confirmation
      const gasPriceNum = parseInt(gasPrice, 16);
      const bufferedGasPrice = Math.floor(gasPriceNum * 1.1);

      console.log(
        `Fetched gas price: ${gasPrice} (${gasPriceNum} wei), buffered: 0x${bufferedGasPrice.toString(
          16
        )}`
      );
      return `0x${bufferedGasPrice.toString(16)}`;
    } catch (error) {
      console.warn("Failed to fetch gas price, using fallback:", error);
      return "0x3b9aca00"; // 1 gwei fallback
    }
  }

  async estimateGas(transactionData, isContractDeployment) {
    try {
      let gasEstimateParams = {
        from: this.walletManager.getAccount(),
        value: transactionData.eth_amount
          ? `0x${parseInt(transactionData.eth_amount).toString(16)}`
          : "0x0",
      };

      if (isContractDeployment && transactionData.bytecode) {
        // For contract deployment
        const bytecode =
          transactionData.bytecode &&
          typeof transactionData.bytecode === "string" &&
          transactionData.bytecode.startsWith("0x")
            ? transactionData.bytecode
            : "0x" + (transactionData.bytecode || "");
        gasEstimateParams.data = bytecode;
      } else {
        // For contract calls
        gasEstimateParams.to =
          transactionData.contract_address || transactionData.token_address;
        gasEstimateParams.data = "0x";
      }

      const estimatedGas = await this.walletManager.selectedWallet.request({
        method: "eth_estimateGas",
        params: [gasEstimateParams],
      });

      // Add 20% buffer to gas limit for safety
      const gasNum = parseInt(estimatedGas, 16);
      const bufferedGas = Math.floor(gasNum * 1.2);

      console.log(
        `Estimated gas: ${estimatedGas} (${gasNum}), buffered: 0x${bufferedGas.toString(
          16
        )}`
      );
      return `0x${bufferedGas.toString(16)}`;
    } catch (error) {
      console.warn("Failed to estimate gas, using fallback:", error);
      // Use higher fallback for contract deployment vs regular transactions
      return isContractDeployment ? "0x2dc6c0" : "0x5208"; // 3M for deployment, 21k for transfer
    }
  }

  async executeTransaction() {
    // Ensure chain is correct
    const targetChainId = this.sessionData.chain_id;
    const currentChainId = this.walletManager.getChainId();

    // Convert target chain ID to hex format if needed
    let targetChainIdHex = targetChainId;
    if (targetChainId && !targetChainId.startsWith("0x")) {
      targetChainIdHex = "0x" + parseInt(targetChainId, 10).toString(16);
    }

    // Only switch network if we have a valid target chain ID and they differ
    if (targetChainIdHex && currentChainId !== targetChainIdHex) {
      await this.walletManager.switchNetwork(targetChainIdHex);
    }

    // Prepare transaction (now async)
    const transactionData = await this.prepareTransactionData();

    // Sign and send transaction
    const txHash = await this.walletManager.signTransaction(transactionData);

    // For deployment transactions, wait for receipt to get contract address
    let contractAddress = null;
    try {
      contractAddress = await this.waitForContractAddress(txHash);
    } catch (error) {
      console.warn("Failed to get contract address from receipt:", error);
    }

    // Update session status
    await this.updateSessionStatus(
      txHash,
      "confirmed",
      contractAddress
    );

    return { txHash, contractAddress };
  }

  async waitForContractAddress(txHash, maxAttempts = 30) {
    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
      try {
        const receipt = await this.walletManager.selectedWallet.request({
          method: "eth_getTransactionReceipt",
          params: [txHash],
        });

        if (receipt) {
          if (receipt.contractAddress) {
            console.log(`Contract address found: ${receipt.contractAddress}`);
            return receipt.contractAddress;
          } else if (receipt.status === "0x0") {
            throw new Error("Transaction failed");
          }
        }
      } catch (error) {
        if (error.message === "Transaction failed") {
          throw error;
        }
      }

      // Wait 2 seconds before next attempt
      await new Promise((resolve) => setTimeout(resolve, 2000));
    }

    throw new Error("Transaction confirmation timeout");
  }

  async updateSessionStatus(txHash, status, contractAddress = null) {
    const requestBody = {
      transaction_hash: txHash,
      status: status,
    };

    if (contractAddress) {
      requestBody.contract_address = contractAddress;
    }

    try {
      let apiUrl = `/api${window.location.pathname}/confirm`;
      console.log(`Updating deployment status at: ${apiUrl}`);

      const response = await fetch(apiUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(requestBody),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      console.log("Deployment status updated successfully:", result);
      return result;
    } catch (error) {
      console.error("Error updating deployment status:", error);
      throw error;
    }
  }

  displayError(message) {
    const contentElement = document.getElementById("content");
    if (contentElement) {
      contentElement.innerHTML = `
                <div class="text-center py-8">
                    <div class="w-16 h-16 mx-auto bg-red-100 rounded-full flex items-center justify-center mb-4">
                        <svg class="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                    </div>
                    <h2 class="text-xl font-semibold text-red-800 mb-2">Error</h2>
                    <p class="text-red-700 text-sm mb-6">${message}</p>
                    <button onclick="location.reload()" class="px-6 py-3 bg-red-100 text-red-700 rounded-xl font-semibold hover:bg-red-200 transition-all duration-200">
                        Try Again
                    </button>
                </div>
            `;
    }
  }
}

// Global token deployment manager
let tokenDeploymentManager;

// Global function for signing transactions
async function signTransaction() {
  const statusElement = document.getElementById("transaction-status");
  const statusMessage = document.getElementById("status-message");
  const signButton = document.getElementById("sign-button");

  // Check if we have the new HTML structure
  const successState = document.getElementById("success-state");
  const contractAddressDisplay = document.getElementById(
    "contract-address-display"
  );
  const transactionHashDisplay = document.getElementById(
    "transaction-hash-display"
  );

  try {
    if (statusElement) {
      statusElement.classList.remove("hidden");
      statusMessage.innerHTML = `
        <div class="flex items-center justify-center">
          <div class="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-500 mr-3"></div>
          Submitting transaction...
        </div>
      `;
    }

    // Execute the transaction
    const result = await tokenDeploymentManager.executeTransaction();

    // If we have the new HTML structure (for deployment page)
    if (successState && contractAddressDisplay) {
      // Hide the main content and show success state
      const content = document.getElementById("content");
      if (content) content.style.display = "none";

      // Update contract address and transaction hash
      if (contractAddressDisplay && result.contractAddress) {
        contractAddressDisplay.textContent = result.contractAddress;
      } else if (contractAddressDisplay) {
        // Check if the showContractAddressUnavailable function exists (from deploy.html)
        if (typeof showContractAddressUnavailable === "function") {
          showContractAddressUnavailable();
        } else {
          contractAddressDisplay.textContent = "Contract address not available";
          contractAddressDisplay.parentElement.classList.add("opacity-50");
        }
      }

      if (transactionHashDisplay) {
        transactionHashDisplay.textContent = result.txHash || result;
      }

      // Show success state with animation
      successState.classList.remove("hidden");
      successState.classList.add("fade-in");

      // Add success glow effect
      const card = successState.closest(".glass-effect");
      if (card) {
        card.classList.add("success-glow");
      }
    } else {
      // Fallback to original display method
      if (statusMessage) {
        statusMessage.innerHTML = `✅ Transaction confirmed! Hash: <code class="bg-green-100 px-2 py-1 rounded text-green-800 text-xs break-all">${result.txHash}</code>`;
      }
      if (statusElement) {
        statusElement.className =
          "bg-green-50 border border-green-200 text-green-800 px-6 py-4 rounded-xl shadow-sm";
      }
    }

    if (signButton) signButton.style.display = "none";
  } catch (error) {
    if (statusMessage)
      statusMessage.innerHTML = `❌ Transaction failed: ${error.message}`;
    if (statusElement) {
      statusElement.className =
        "bg-red-50 border border-red-200 text-red-800 px-6 py-4 rounded-xl shadow-sm";
    }
  }
}

// Initialize on page load
document.addEventListener("DOMContentLoaded", function () {
  // Only initialize if walletManager exists (from wallet-connection.js)
  if (typeof walletManager !== "undefined") {
    tokenDeploymentManager = new TokenDeploymentManager(walletManager);

    // Load session data if available
    const sessionData = document.getElementById("session-data");
    if (sessionData) {
      const sessionId = sessionData.dataset.sessionId;
      const apiUrl = sessionData.dataset.apiUrl;
      const embeddedTransactionData = sessionData.dataset.transactionData;

      if (sessionId && apiUrl) {
        // Parse embedded transaction data if available
        let parsedEmbeddedData = null;
        if (embeddedTransactionData) {
          try {
            parsedEmbeddedData = JSON.parse(embeddedTransactionData);
            console.log("Found embedded transaction data:", parsedEmbeddedData);
          } catch (error) {
            console.error("Error parsing embedded transaction data:", error);
          }
        }

        tokenDeploymentManager.loadSessionData(
          sessionId,
          apiUrl,
          parsedEmbeddedData
        );
      }
    }
  }
});
