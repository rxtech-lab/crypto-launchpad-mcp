// Balance Query Management
class BalanceQueryManager {
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
      this.handleBalanceQuery();
    } catch (error) {
      console.error("Error loading session data:", error);
      this.displayError("Failed to load balance details");
    }
  }

  async handleBalanceQuery() {
    console.log("Handling balance query session");
    
    // Always initialize the wallet interface first for balance queries
    await this.initializeBalanceSession();
    
    // Check if we already have balance data from the server
    if (this.sessionData.balance_data) {
      // Display the balance data immediately
      if (typeof updateBalance === 'function') {
        updateBalance(this.sessionData.balance_data);
      }
      return;
    }

    // Check if a wallet address is already provided
    const walletAddress = this.sessionData.wallet_address;
    
    if (walletAddress && walletAddress !== "") {
      // Auto-fetch balance data if wallet address is provided
      // But still show the wallet interface for manual refresh option
      setTimeout(() => {
        this.fetchBalanceData(walletAddress);
      }, 500);
    }
  }

  async initializeBalanceSession() {
    const contentElement = document.getElementById("content");
    if (!contentElement) return;

    contentElement.innerHTML = `
      <div class="max-w-2xl mx-auto space-y-6 p-6">
        <!-- Header -->
        <div class="text-center mb-8">
          <div class="w-16 h-16 mx-auto bg-blue-100 rounded-full flex items-center justify-center mb-4">
            <svg class="w-8 h-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z"></path>
            </svg>
          </div>
          <h2 class="text-xl font-semibold text-gray-900 mb-2">Connect Your Wallet</h2>
          <p class="text-gray-600 text-sm mb-6">Connect your wallet to view your balance</p>
        </div>

        <!-- Wallet Selection -->
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

        <!-- Fetch Balance Button (replaces sign button for balance queries) -->
        <button id="sign-button" onclick="fetchBalanceFromWallet()" style="display: none;" class="w-full bg-green-600 text-white py-4 px-6 rounded-xl font-medium hover:bg-green-700 active:bg-green-800 transition-all transform hover:scale-[1.02] active:scale-[0.98] shadow-lg">
          Fetch Balance
        </button>
      </div>
    `;

    // Update wallet manager to show correct UI elements
    this.walletManager.updateConnectionStatus();
  }

  async fetchBalanceData(walletAddress) {
    try {
      // Make API call to get balance data
      const response = await fetch(`${this.apiUrl}?wallet_address=${encodeURIComponent(walletAddress)}`);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      const data = await response.json();
      
      if (data.balance_data) {
        // Update the balance display
        if (typeof updateBalance === 'function') {
          updateBalance(data.balance_data);
        }
      } else if (data.balance_error) {
        throw new Error(data.balance_error);
      }
    } catch (error) {
      console.error("Failed to fetch balance data:", error);
      
      // Show error state
      const contentElement = document.getElementById("content");
      if (contentElement) {
        contentElement.innerHTML = `
          <div class="text-center py-8">
            <div class="w-16 h-16 mx-auto bg-red-100 rounded-full flex items-center justify-center mb-4">
              <svg class="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
            </div>
            <h2 class="text-xl font-semibold text-red-800 mb-2">Error Loading Balance</h2>
            <p class="text-red-700 text-sm mb-6">${error.message}</p>
            <button onclick="location.reload()" class="px-6 py-3 bg-red-100 text-red-700 rounded-xl font-semibold hover:bg-red-200 transition-all duration-200">
              Try Again
            </button>
          </div>
        `;
      }
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

// Global balance query manager
let balanceQueryManager;

// Global function for balance query
async function fetchBalanceFromWallet() {
  try {
    if (!walletManager || !walletManager.account) {
      alert("Please connect your wallet first");
      return;
    }

    const walletAddress = walletManager.account;
    
    // Update button state
    const button = document.getElementById('sign-button');
    if (button) {
      const originalHTML = button.innerHTML;
      button.innerHTML = '<div class="flex items-center justify-center"><div class="animate-spin rounded-full h-5 w-5 border-b-2 border-white mr-3"></div>Fetching Balance...</div>';
      button.disabled = true;
    }

    // Fetch balance data
    await balanceQueryManager.fetchBalanceData(walletAddress);

  } catch (error) {
    console.error("Failed to fetch balance:", error);
    alert("Failed to fetch balance: " + error.message);
    
    // Reset button state
    const button = document.getElementById('sign-button');
    if (button) {
      button.innerHTML = 'Fetch Balance';
      button.disabled = false;
    }
  }
}

// Initialize on page load
document.addEventListener("DOMContentLoaded", function () {
  // Only initialize if walletManager exists (from wallet-connection.js)
  if (typeof walletManager !== 'undefined') {
    balanceQueryManager = new BalanceQueryManager(walletManager);

    // Load session data if available
    const sessionData = document.getElementById("session-data");
    if (sessionData) {
      const sessionId = sessionData.dataset.sessionId;
      const apiUrl = sessionData.dataset.apiUrl;

      if (sessionId && apiUrl) {
        balanceQueryManager.loadSessionData(sessionId, apiUrl);
      }
    }
  }
});