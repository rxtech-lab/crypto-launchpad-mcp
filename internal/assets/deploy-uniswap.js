// Uniswap Deployment Management
class UniswapDeploymentManager {
  constructor(walletManager) {
    this.walletManager = walletManager;
    this.sessionData = null;
    this.apiUrl = null;
  }

  async loadSessionData(sessionId, apiUrl) {
    this.apiUrl = apiUrl;
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

    const { deployment_data, session_type, metadata } = this.sessionData;

    if (session_type !== "deploy_uniswap") {
      console.error("Invalid session type for Uniswap deployment:", session_type);
      this.displayError("Invalid session type");
      return;
    }

    const detailsHTML = this.generateUniswapDeploymentDetails({
      deployment_data,
      metadata
    });

    contentElement.innerHTML = `
            <div class="max-w-2xl mx-auto space-y-6 p-6">
                <!-- Header -->
                <div class="text-center mb-8">
                    <div class="w-16 h-16 mx-auto bg-purple-100 rounded-full flex items-center justify-center mb-4">
                        <svg class="w-8 h-8 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"></path>
                        </svg>
                    </div>
                    <h2 class="text-xl font-semibold text-gray-900 mb-2">Deploy Uniswap Infrastructure</h2>
                    <p class="text-gray-600 text-sm mb-6">Review and confirm your Uniswap deployment</p>
                </div>

                <!-- Transaction Details -->
                <div class="bg-gray-50 rounded-2xl p-6 mb-6">
                    <h3 class="text-lg font-semibold text-gray-900 mb-4">Deployment Details</h3>
                    ${detailsHTML}
                </div>

                <!-- Wallet Connection -->
                <div class="space-y-4">
                    <div>
                        <label for="wallet-select" class="block text-sm font-medium text-gray-700 mb-2">Select Wallet</label>
                        <select id="wallet-select" class="w-full px-4 py-3 bg-white border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent">
                            <option value="">Loading wallets...</option>
                        </select>
                    </div>
                    
                    <button id="connect-button" onclick="connectWallet()" class="w-full bg-purple-600 text-white py-4 px-6 rounded-xl font-medium hover:bg-purple-700 active:bg-purple-800 transition-all transform hover:scale-[1.02] active:scale-[0.98] shadow-lg">
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
                <button id="sign-button" onclick="signUniswapDeployment()" style="display: none;" class="w-full bg-green-600 text-white py-4 px-6 rounded-xl font-medium hover:bg-green-700 active:bg-green-800 transition-all transform hover:scale-[1.02] active:scale-[0.98] shadow-lg">
                    Deploy Uniswap Contracts
                </button>
            </div>
        `;

    // Update wallet connection status
    this.walletManager.updateConnectionStatus();
  }

  generateUniswapDeploymentDetails(data) {
    const deploymentData = data.deployment_data;
    const metadata = data.metadata || [];
    
    let metadataHTML = '';
    metadata.forEach(item => {
      metadataHTML += `
        <div class="bg-gray-50 rounded-xl p-4">
          <span class="text-gray-600 text-sm font-medium">${item.title}</span>
          <div class="font-semibold text-gray-900 text-lg mt-1">${item.value}</div>
        </div>
      `;
    });
    
    return `
      <div class="space-y-4">
        <!-- Deployment Overview -->
        <div class="bg-purple-50 border border-purple-200 rounded-xl p-4">
          <h3 class="font-semibold text-purple-900 mb-2">Uniswap V2 Infrastructure Deployment</h3>
          <p class="text-purple-700 text-sm">This will deploy 3 contracts in sequence: WETH9, Factory, and Router</p>
        </div>
        
        <!-- Metadata Grid -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          ${metadataHTML}
        </div>
        
        <!-- Contract List -->
        <div class="space-y-3">
          <h4 class="font-medium text-gray-900">Contracts to Deploy:</h4>
          <div class="space-y-2">
            <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
              <span class="font-medium">WETH9</span>
              <span class="text-sm text-gray-600">Wrapped ETH Token</span>
            </div>
            <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
              <span class="font-medium">Factory</span>
              <span class="text-sm text-gray-600">Pair Creation Contract</span>
            </div>
            <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
              <span class="font-medium">Router</span>
              <span class="text-sm text-gray-600">Liquidity & Swap Contract</span>
            </div>
          </div>
        </div>
        
        <!-- Gas Estimate -->
        <div class="bg-blue-50 border border-blue-200 rounded-xl p-4">
          <span class="text-blue-600 text-sm font-medium">Estimated Gas Cost</span>
          <div class="font-semibold text-blue-900 text-lg mt-1">~0.1 ETH total</div>
          <p class="text-blue-700 text-xs mt-1">Actual cost depends on network conditions</p>
        </div>
      </div>
    `;
  }

  async executeUniswapDeployment() {
    const { deployment_data } = this.sessionData;
    
    // For now, return mock data to test the interface
    // In a real implementation, this would deploy each contract in sequence
    const mockResult = {
      success: true,
      contractAddresses: {
        weth: "0x1234567890123456789012345678901234567890",
        factory: "0x2345678901234567890123456789012345678901", 
        router: "0x3456789012345678901234567890123456789012"
      },
      transactionHashes: {
        weth: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
        factory: "0xbcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890a",
        router: "0xcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
      }
    };

    // Update session with multiple contract addresses and transaction hashes
    await this.updateUniswapSessionStatus(mockResult);
    
    return mockResult;
  }

  async updateUniswapSessionStatus(deploymentResult) {
    const requestBody = {
      transaction_hashes: deploymentResult.transactionHashes,
      contract_addresses: deploymentResult.contractAddresses,
      deployer_address: this.walletManager.getAccount(),
      status: "confirmed"
    };

    try {
      let apiUrl = `/api${window.location.pathname}/confirm`;
      console.log(`Updating Uniswap deployment status at: ${apiUrl}`);

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
      console.log("Uniswap deployment status updated successfully:", result);
      return result;
    } catch (error) {
      console.error("Failed to update Uniswap deployment status:", error);
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

// Global uniswap deployment manager
let uniswapDeploymentManager;

// Global function for signing Uniswap deployment
async function signUniswapDeployment() {
  const statusElement = document.getElementById("transaction-status");
  const statusMessage = document.getElementById("status-message");
  const signButton = document.getElementById("sign-button");

  // Check if we have the new HTML structure
  const successState = document.getElementById("success-state");

  try {
    if (statusElement) {
      statusElement.classList.remove("hidden");
      statusMessage.innerHTML = `
        <div class="flex items-center justify-center">
          <div class="animate-spin rounded-full h-5 w-5 border-b-2 border-purple-500 mr-3"></div>
          Deploying Uniswap contracts...
        </div>
      `;
    }

    // Execute the Uniswap deployment
    const result = await uniswapDeploymentManager.executeUniswapDeployment();

    // If we have the new HTML structure (for deployment page)
    if (successState) {
      // Hide the main content and show success state
      const content = document.getElementById("content");
      if (content) content.style.display = "none";

      // Update Uniswap contract addresses
      if (result.contractAddresses) {
        const wethAddress = document.getElementById("weth-address");
        const factoryAddress = document.getElementById("factory-address");
        const routerAddress = document.getElementById("router-address");
        
        if (wethAddress && result.contractAddresses.weth) {
          wethAddress.textContent = result.contractAddresses.weth;
        }
        if (factoryAddress && result.contractAddresses.factory) {
          factoryAddress.textContent = result.contractAddresses.factory;
        }
        if (routerAddress && result.contractAddresses.router) {
          routerAddress.textContent = result.contractAddresses.router;
        }
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
        statusMessage.innerHTML = `✅ Uniswap deployment confirmed!`;
      }
      if (statusElement) {
        statusElement.className = "bg-green-50 border border-green-200 text-green-800 px-6 py-4 rounded-xl shadow-sm";
      }
    }

    if (signButton) signButton.style.display = "none";
  } catch (error) {
    if (statusMessage)
      statusMessage.innerHTML = `❌ Deployment failed: ${error.message}`;
    if (statusElement) {
      statusElement.className = "bg-red-50 border border-red-200 text-red-800 px-6 py-4 rounded-xl shadow-sm";
    }
  }
}

// Initialize on page load
document.addEventListener("DOMContentLoaded", function () {
  // Only initialize if walletManager exists (from wallet-connection.js)
  if (typeof walletManager !== 'undefined') {
    uniswapDeploymentManager = new UniswapDeploymentManager(walletManager);

    // Load session data if available
    const sessionData = document.getElementById("session-data");
    if (sessionData) {
      const sessionId = sessionData.dataset.sessionId;
      const apiUrl = sessionData.dataset.apiUrl;

      if (sessionId && apiUrl) {
        uniswapDeploymentManager.loadSessionData(sessionId, apiUrl);
      }
    }
  }
});