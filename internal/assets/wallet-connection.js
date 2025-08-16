// EIP-6963 Wallet Discovery and Connection Management
class WalletManager {
  constructor() {
    this.wallets = new Map();
    this.selectedWallet = null;
    this.account = null;
    this.chainId = null;
    this.setupWalletDiscovery();
  }

  setupWalletDiscovery() {
    // Listen for EIP-6963 wallet announcements
    window.addEventListener("eip6963:announceProvider", (event) => {
      const { info, provider } = event.detail;
      this.wallets.set(info.uuid, { info, provider });
      this.updateWalletList();
    });

    // Request wallet announcements
    window.dispatchEvent(new Event("eip6963:requestProvider"));
  }

  updateWalletList(retryCount = 0) {
    console.log("Updating wallet list...");
    const walletSelect = document.getElementById("wallet-select");
    if (!walletSelect) {
      if (retryCount < 10) { // Limit retries to prevent infinite loop
        console.warn(`wallet-select element not found - retry ${retryCount + 1}/10`);
        // Retry after a short delay in case the element is being created
        setTimeout(() => this.updateWalletList(retryCount + 1), 100);
      } else {
        console.warn("wallet-select element not found after 10 retries");
      }
      return;
    }

    console.log("Found", this.wallets.size, "wallets");
    walletSelect.innerHTML = '<option value="">Select a wallet...</option>';
    console.log("Wallet Select Element", walletSelect);

    for (const [uuid, wallet] of this.wallets) {
      console.log("Adding wallet to list:", wallet.info.name);
      const option = document.createElement("option");
      option.value = uuid;
      option.textContent = wallet.info.name;
      walletSelect.appendChild(option);
    }

    console.log(
      "Wallet list updated successfully with",
      this.wallets.size,
      "wallets"
    );

    if (this.wallets.size > 0) {
      console.log(
        "Wallets available:",
        Array.from(this.wallets.values()).map((w) => w.info.name)
      );
    }
  }

  async connectWallet(walletUuid) {
    const wallet = this.wallets.get(walletUuid);
    if (!wallet) {
      throw new Error("Wallet not found");
    }

    try {
      console.log("Connecting to wallet:", wallet.info.name);
      this.selectedWallet = wallet.provider;

      // Request account access
      const accounts = await this.selectedWallet.request({
        method: "eth_requestAccounts",
      });

      if (accounts.length === 0) {
        throw new Error("No accounts available");
      }

      this.account = accounts[0];
      console.log("Connected to account:", this.account);

      // Get chain ID
      this.chainId = await this.selectedWallet.request({
        method: "eth_chainId",
      });

      console.log("Chain ID:", this.chainId);

      // Update UI
      this.updateConnectionStatus();

      return {
        account: this.account,
        chainId: this.chainId,
      };
    } catch (error) {
      console.error("Failed to connect wallet:", error);
      throw error;
    }
  }

  async switchNetwork(targetChainId) {
    if (!this.selectedWallet) {
      throw new Error("No wallet connected");
    }

    try {
      await this.selectedWallet.request({
        method: "wallet_switchEthereumChain",
        params: [{ chainId: targetChainId }],
      });

      this.chainId = targetChainId;
      this.updateConnectionStatus();
    } catch (error) {
      console.error("Failed to switch network:", error);
      throw error;
    }
  }

  async signTransaction(transactionData) {
    if (!this.selectedWallet || !this.account) {
      throw new Error("Wallet not connected");
    }

    try {
      const txHash = await this.selectedWallet.request({
        method: "eth_sendTransaction",
        params: [transactionData],
      });

      return txHash;
    } catch (error) {
      console.error("Transaction failed:", error);
      throw error;
    }
  }

  updateConnectionStatus() {
    const statusElement = document.getElementById("connection-status");
    const connectButton = document.getElementById("connect-button");
    const signButton = document.getElementById("sign-button");

    if (this.account) {
      if (statusElement) {
        statusElement.innerHTML = `
                    <div class="bg-green-50 border border-green-200 rounded-2xl p-5">
                        <div class="flex items-center">
                            <div class="w-10 h-10 bg-green-100 rounded-full flex items-center justify-center mr-4">
                                <svg class="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                                </svg>
                            </div>
                            <div class="flex-1">
                                <p class="font-medium text-green-800 mb-1">Wallet Connected</p>
                                <div class="space-y-1">
                                    <div class="flex justify-between text-sm">
                                        <span class="text-green-700">Address:</span>
                                        <code class="bg-green-100 px-2 py-1 rounded text-green-800">${this.account.slice(
                                          0,
                                          6
                                        )}...${this.account.slice(-4)}</code>
                                    </div>
                                    <div class="flex justify-between text-sm">
                                        <span class="text-green-700">Chain ID:</span>
                                        <span class="text-green-800 font-medium">${parseInt(
                                          this.chainId,
                                          16
                                        )}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                `;
      }
      if (connectButton) connectButton.style.display = "none";
      if (signButton) signButton.style.display = "block";
    } else {
      if (statusElement) {
        statusElement.innerHTML = `
                    <div class="bg-amber-50 border border-amber-200 rounded-2xl p-5">
                        <div class="flex items-center">
                            <div class="w-10 h-10 bg-amber-100 rounded-full flex items-center justify-center mr-4">
                                <svg class="w-5 h-5 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16c-.77.833.192 2.5 1.732 2.5z"></path>
                                </svg>
                            </div>
                            <div class="flex-1">
                                <p class="font-medium text-amber-800 mb-1">Wallet Required</p>
                                <p class="text-sm text-amber-700">Please connect your wallet to continue</p>
                            </div>
                        </div>
                    </div>
                `;
      }
      if (connectButton) connectButton.style.display = "block";
      if (signButton) signButton.style.display = "none";
    }
  }

  getAccount() {
    return this.account;
  }

  getChainId() {
    return this.chainId;
  }

  isConnected() {
    return this.account !== null;
  }
}

// Global wallet manager instance
let walletManager;

// Global function for connecting wallet
async function connectWallet() {
  const walletSelect = document.getElementById("wallet-select");
  const selectedWallet = walletSelect.value;

  if (!selectedWallet) {
    alert("Please select a wallet");
    return;
  }

  try {
    await walletManager.connectWallet(selectedWallet);
  } catch (error) {
    alert("Failed to connect wallet: " + error.message);
  }
}

// Initialize wallet manager on page load
document.addEventListener("DOMContentLoaded", function () {
  walletManager = new WalletManager();
});