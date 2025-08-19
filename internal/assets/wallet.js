// EIP-6963 Wallet Discovery and Transaction Signing
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
      if (retryCount < 10) {
        // Limit retries to prevent infinite loop
        console.warn(
          `wallet-select element not found - retry ${retryCount + 1}/10`
        );
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
        "Wallet list updated successfully with",
        this.wallets.size,
        "wallets"
      );
    } else {
      console.warn("No wallets available to display");
    }
  }

  async connectWallet(walletUuid) {
    if (!this.wallets.has(walletUuid)) {
      throw new Error("Wallet not found");
    }

    const wallet = this.wallets.get(walletUuid);
    this.selectedWallet = wallet.provider;

    try {
      // Request account access
      const accounts = await this.selectedWallet.request({
        method: "eth_requestAccounts",
      });

      if (accounts.length === 0) {
        throw new Error("No accounts available");
      }

      this.account = accounts[0];

      // Get current chain ID
      this.chainId = await this.selectedWallet.request({
        method: "eth_chainId",
      });

      this.updateConnectionStatus();
      return { account: this.account, chainId: this.chainId };
    } catch (error) {
      console.error("Failed to connect wallet:", error);
      throw error;
    }
  }

  async switchNetwork(targetChainId) {
    if (!this.selectedWallet) {
      throw new Error("No wallet connected");
    }

    const chainIdHex = "0x" + parseInt(targetChainId).toString(16);

    try {
      await this.selectedWallet.request({
        method: "wallet_switchEthereumChain",
        params: [{ chainId: chainIdHex }],
      });

      this.chainId = chainIdHex;
      this.updateConnectionStatus();
    } catch (error) {
      // If the chain hasn't been added to the wallet, add it
      if (error.code === 4902) {
        await this.addNetwork(targetChainId);
      } else {
        throw error;
      }
    }
  }

  async addNetwork(chainId) {
    const networks = {
      1: {
        chainId: "0x1",
        chainName: "Ethereum Mainnet",
        rpcUrls: ["https://ethereum.publicnode.com"],
        nativeCurrency: { name: "ETH", symbol: "ETH", decimals: 18 },
        blockExplorerUrls: ["https://etherscan.io"],
      },
      11155111: {
        chainId: "0xaa36a7",
        chainName: "Sepolia",
        rpcUrls: [
          "https://sepolia.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161",
        ],
        nativeCurrency: { name: "SepoliaETH", symbol: "ETH", decimals: 18 },
        blockExplorerUrls: ["https://sepolia.etherscan.io"],
      },
    };

    const networkConfig = networks[chainId];
    if (!networkConfig) {
      throw new Error("Network configuration not found");
    }

    await this.selectedWallet.request({
      method: "wallet_addEthereumChain",
      params: [networkConfig],
    });

    this.chainId = networkConfig.chainId;
    this.updateConnectionStatus();
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

  isConnected() {
    return this.selectedWallet && this.account;
  }

  getAccount() {
    return this.account;
  }

  getChainId() {
    return this.chainId;
  }
}

// Transaction Manager for handling different transaction types
class TransactionManager {
  constructor(walletManager) {
    this.walletManager = walletManager;
    this.sessionData = null;
  }

  async loadSessionData(sessionId, apiUrl) {
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

    const { transaction_data, session_type, chain_type, chain_id } =
      this.sessionData;

    let detailsHTML = "";

    switch (session_type) {
      case "deploy":
        detailsHTML = this.generateDeploymentDetails(transaction_data);
        break;
      case "deploy_uniswap":
        detailsHTML = this.generateUniswapDeploymentDetails(transaction_data);
        break;
      case "create_pool":
        detailsHTML = this.generateCreatePoolDetails(transaction_data);
        break;
      case "add_liquidity":
        detailsHTML = this.generateAddLiquidityDetails(transaction_data);
        break;
      case "remove_liquidity":
        detailsHTML = this.generateRemoveLiquidityDetails(transaction_data);
        break;
      case "swap":
        detailsHTML = this.generateSwapDetails(transaction_data);
        break;
      case "balance_query":
        // Balance query doesn't need transaction details, handle it differently
        this.handleBalanceQuery();
        return;
      default:
        detailsHTML = '<p class="text-red-600">Unknown transaction type</p>';
    }

    contentElement.innerHTML = `
            <div class="max-w-2xl mx-auto space-y-6 p-6">
                <!-- Header -->
                <div class="text-center mb-8">
                    <h1 class="text-2xl font-semibold text-gray-900 mb-2">Transaction Signing</h1>
                    <p class="text-gray-600">Review and confirm your transaction</p>
                </div>

                <!-- Transaction Details Card -->
                <div class="bg-white border border-gray-200 rounded-2xl p-6 shadow-sm">
                    <div class="flex items-center mb-4">
                        <div class="w-10 h-10 bg-blue-100 rounded-full flex items-center justify-center mr-3">
                            <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                            </svg>
                        </div>
                        <h2 class="text-lg font-medium text-gray-900">Transaction Details</h2>
                    </div>
                    <div class="space-y-3">
                        ${detailsHTML}
                    </div>
                </div>

                <!-- Network Info Card -->
                <div class="bg-gray-50 border border-gray-200 rounded-2xl p-5">
                    <div class="flex items-center mb-3">
                        <div class="w-8 h-8 bg-gray-200 rounded-full flex items-center justify-center mr-3">
                            <svg class="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9v-9m0-9v9"></path>
                            </svg>
                        </div>
                        <h3 class="text-base font-medium text-gray-800">Network</h3>
                    </div>
                    <div class="flex justify-between py-2">
                        <span class="text-gray-600">Chain</span>
                        <span class="font-medium text-gray-900">${
                          chain_type.charAt(0).toUpperCase() +
                          chain_type.slice(1)
                        }</span>
                    </div>
                    <div class="flex justify-between py-2 border-t border-gray-200">
                        <span class="text-gray-600">Chain ID</span>
                        <span class="font-medium text-gray-900">${chain_id}</span>
                    </div>
                </div>

                <!-- Connection Status -->
                <div id="connection-status"></div>

                <!-- Wallet Connection -->
                <div class="space-y-4">
                    <div id="connect-button">
                        <div class="space-y-3">
                            <select id="wallet-select" class="w-full px-4 py-3 border border-gray-300 rounded-xl bg-white text-gray-900 focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all">
                                <option value="">Choose your wallet...</option>
                            </select>
                            <button onclick="connectWallet()" class="w-full bg-blue-600 text-white py-3 px-6 rounded-xl font-medium hover:bg-blue-700 active:bg-blue-800 transition-all transform hover:scale-[1.02] active:scale-[0.98] shadow-sm">
                                Connect Wallet
                            </button>
                        </div>
                    </div>

                    <button id="sign-button" onclick="signTransaction()" style="display: none;" class="w-full bg-green-600 text-white py-4 px-6 rounded-xl font-medium hover:bg-green-700 active:bg-green-800 transition-all transform hover:scale-[1.02] active:scale-[0.98] shadow-lg">
                        <div class="flex items-center justify-center">
                            <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
                            </svg>
                            Sign & Send Transaction
                        </div>
                    </button>
                </div>

                <!-- Transaction Status -->
                <div id="transaction-status" class="hidden rounded-2xl p-6">
                    <div class="text-center">
                        <p id="status-message" class="text-base">Processing transaction...</p>
                    </div>
                </div>
            </div>
        `;

    // Update connection status first
    this.walletManager.updateConnectionStatus();

    // Now that the DOM is ready, update the wallet list
    this.walletManager.updateWalletList();
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

  generateUniswapDeploymentDetails(data) {
    const deploymentData = data.deployment_data;
    const metadata = data.metadata || [];

    let metadataHTML = "";
    metadata.forEach((item) => {
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

  generateCreatePoolDetails(data) {
    return `
            <div class="space-y-3">
                <div class="py-2">
                    <span class="text-gray-600 block mb-1">Token Address</span>
                    <code class="text-xs bg-gray-100 px-2 py-1 rounded-md text-gray-800 break-all">${
                      data.token_address
                    }</code>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">Token Amount</span>
                    <span class="font-medium text-gray-900">${
                      data.initial_token_amount
                    }</span>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">ETH Amount</span>
                    <span class="font-medium text-gray-900">${
                      data.initial_eth_amount
                    } ETH</span>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">Protocol</span>
                    <span class="font-medium text-purple-600">${data.uniswap_version.toUpperCase()}</span>
                </div>
                <div class="py-2 border-t border-gray-100">
                    <span class="text-gray-600 block mb-1">Creator</span>
                    <code class="text-xs bg-gray-100 px-2 py-1 rounded-md text-gray-800 break-all">${
                      data.creator_address
                    }</code>
                </div>
            </div>
        `;
  }

  generateAddLiquidityDetails(data) {
    return `
            <div class="space-y-3">
                <div class="py-2">
                    <span class="text-gray-600 block mb-1">Token Address</span>
                    <code class="text-xs bg-gray-100 px-2 py-1 rounded-md text-gray-800 break-all">${data.token_address}</code>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">Token Amount</span>
                    <span class="font-medium text-gray-900">${data.token_amount}</span>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">ETH Amount</span>
                    <span class="font-medium text-gray-900">${data.eth_amount} ETH</span>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">Min Token</span>
                    <span class="text-sm text-gray-700">${data.min_token_amount}</span>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">Min ETH</span>
                    <span class="text-sm text-gray-700">${data.min_eth_amount} ETH</span>
                </div>
                <div class="py-2 border-t border-gray-100">
                    <span class="text-gray-600 block mb-1">User Address</span>
                    <code class="text-xs bg-gray-100 px-2 py-1 rounded-md text-gray-800 break-all">${data.user_address}</code>
                </div>
            </div>
        `;
  }

  generateRemoveLiquidityDetails(data) {
    return `
            <div class="space-y-3">
                <div class="py-2">
                    <span class="text-gray-600 block mb-1">Token Address</span>
                    <code class="text-xs bg-gray-100 px-2 py-1 rounded-md text-gray-800 break-all">${data.token_address}</code>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">Liquidity Amount</span>
                    <span class="font-medium text-red-600">${data.liquidity_amount}</span>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">Min Token</span>
                    <span class="text-sm text-gray-700">${data.min_token_amount}</span>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">Min ETH</span>
                    <span class="text-sm text-gray-700">${data.min_eth_amount} ETH</span>
                </div>
                <div class="py-2 border-t border-gray-100">
                    <span class="text-gray-600 block mb-1">User Address</span>
                    <code class="text-xs bg-gray-100 px-2 py-1 rounded-md text-gray-800 break-all">${data.user_address}</code>
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
                        <div class="flex flex-col items-center">
                            <span class="text-sm text-gray-600 mb-1">From</span>
                            <span class="font-mono text-sm bg-white px-3 py-1 rounded-lg">${
                              fromDisplay === "ETH"
                                ? "ETH"
                                : fromDisplay.slice(0, 8) + "..."
                            }</span>
                        </div>
                        <div class="px-4">
                            <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3"></path>
                            </svg>
                        </div>
                        <div class="flex flex-col items-center">
                            <span class="text-sm text-gray-600 mb-1">To</span>
                            <span class="font-mono text-sm bg-white px-3 py-1 rounded-lg">${
                              toDisplay === "ETH"
                                ? "ETH"
                                : toDisplay.slice(0, 8) + "..."
                            }</span>
                        </div>
                    </div>
                </div>
                <div class="flex justify-between items-center py-2">
                    <span class="text-gray-600">Amount</span>
                    <span class="font-medium text-gray-900">${
                      data.amount
                    }</span>
                </div>
                <div class="flex justify-between items-center py-2 border-t border-gray-100">
                    <span class="text-gray-600">Slippage</span>
                    <span class="text-orange-600 font-medium">${
                      data.slippage_tolerance
                    }%</span>
                </div>
                <div class="py-2 border-t border-gray-100">
                    <span class="text-gray-600 block mb-1">User Address</span>
                    <code class="text-xs bg-gray-100 px-2 py-1 rounded-md text-gray-800 break-all">${
                      data.user_address
                    }</code>
                </div>
            </div>
        `;
  }

  displayError(message) {
    const contentElement = document.getElementById("content");
    if (contentElement) {
      contentElement.innerHTML = `
                <div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
                    <p>${message}</p>
                </div>
            `;
    }
  }

  async executeTransaction() {
    if (!this.walletManager.isConnected()) {
      throw new Error("Wallet not connected");
    }

    if (!this.sessionData) {
      throw new Error("No transaction data loaded");
    }

    // Check if we're on the correct network
    const targetChainId = this.sessionData.chain_id;
    const currentChainId = parseInt(
      this.walletManager.getChainId(),
      16
    ).toString();

    if (currentChainId !== targetChainId) {
      await this.walletManager.switchNetwork(targetChainId);
    }

    // Handle Uniswap deployment differently
    if (this.sessionData.session_type === "deploy_uniswap") {
      return await this.executeUniswapDeployment();
    }

    // Prepare transaction based on type
    const transactionData = this.prepareTransactionData();

    // Sign and send transaction
    const txHash = await this.walletManager.signTransaction(transactionData);

    // For deployment transactions, wait for receipt to get contract address
    let contractAddress = null;
    if (this.sessionData.session_type === "deploy") {
      try {
        contractAddress = await this.waitForContractAddress(txHash);
      } catch (error) {
        console.warn("Failed to get contract address from receipt:", error);
        // Don't throw error here, still update session with txHash
      }
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
    // Wait for transaction receipt with contract address
    // Try every 2 seconds for up to 60 seconds (30 attempts)

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
            // Transaction failed
            throw new Error("Transaction failed during execution");
          } else {
            // Transaction succeeded but no contract address (shouldn't happen for deployments)
            console.warn("Transaction succeeded but no contract address found");
            return null;
          }
        }

        // Receipt not available yet, wait and try again
        if (attempt < maxAttempts) {
          console.log(
            `Waiting for transaction confirmation... (attempt ${attempt}/${maxAttempts})`
          );
          await new Promise((resolve) => setTimeout(resolve, 2000));
        }
      } catch (error) {
        if (attempt === maxAttempts) {
          throw new Error(
            `Failed to get transaction receipt after ${maxAttempts} attempts: ${error.message}`
          );
        }

        // For non-final attempts, just log and continue
        console.warn(`Attempt ${attempt} failed:`, error.message);
        await new Promise((resolve) => setTimeout(resolve, 2000));
      }
    }

    throw new Error(
      "Transaction confirmation timeout - please check the blockchain explorer"
    );
  }

  prepareTransactionData() {
    const { transaction_data, session_type } = this.sessionData;

    // Handle Uniswap deployment differently
    if (session_type === "deploy_uniswap") {
      return this.prepareUniswapDeploymentData();
    }

    // This is a simplified transaction preparation for regular deployments
    // In a real implementation, this would use proper contract ABIs and encoding
    return {
      from: this.walletManager.getAccount(),
      to: transaction_data.contract_address || transaction_data.token_address,
      value: transaction_data.eth_amount
        ? `0x${parseInt(transaction_data.eth_amount).toString(16)}`
        : "0x0",
      gas: "0x186a0", // 100000 gas limit
      gasPrice: "0x9184e72a000", // 10 gwei
      data: "0x", // Contract data would be encoded here
    };
  }

  prepareUniswapDeploymentData() {
    const { deployment_data } = this.sessionData;

    // For Uniswap deployment, we return the deployment data structure
    // This will be handled by custom deployment logic
    return {
      type: "uniswap_deployment",
      contracts: deployment_data || {},
      from: this.walletManager.getAccount(),
      gas: "0x186a0", // 100000 gas limit per contract
      gasPrice: "0x9184e72a000", // 10 gwei
    };
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
        router: "0x3456789012345678901234567890123456789012",
      },
      transactionHashes: {
        weth: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
        factory:
          "0xbcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890a",
        router:
          "0xcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
      },
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
      status: "confirmed",
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

  async updateSessionStatus(txHash, status, contractAddress = null) {
    // Construct the confirm URL by appending /confirm to current path
    const requestBody = {
      transaction_hash: txHash,
      status: status,
    };

    if (contractAddress) {
      requestBody.contract_address = contractAddress;
    }

    try {
      // If we're on /deploy/:session_id, this will create /api/deploy/:session_id/confirm
      let apiUrl = `/api${window.location.pathname}/confirm`;

      console.log(`Updating deployment status at: ${apiUrl}`);
      console.log(`Request body:`, requestBody);

      const response = await fetch(apiUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(requestBody),
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error(`HTTP ${response.status}: ${errorText}`);
        throw new Error(
          `Failed to update deployment status: ${response.status} ${errorText}`
        );
      }

      const result = await response.json();
      console.log("Deployment status updated successfully:", result);

      // Log deployment status update for confirmation
      if (
        this.sessionData?.session_type === "deploy" &&
        status === "confirmed"
      ) {
        console.log("✅ Deployment record updated in database:");
        console.log(`   - Transaction Hash: ${txHash}`);
        if (contractAddress) {
          console.log(`   - Contract Address: ${contractAddress}`);
        }
        console.log(`   - Status: ${status}`);
      }

      return result;
    } catch (error) {
      console.error("Error updating deployment status:", error);
      throw error;
    }
  }

  async handleBalanceQuery() {
    console.log("Handling balance query session");

    // Always initialize the wallet interface first for balance queries
    await this.initializeBalanceSession();

    // Check if we already have balance data from the server
    if (this.sessionData.balance_data) {
      // Display the balance data immediately
      if (typeof updateBalance === "function") {
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
      const response = await fetch(
        `${this.apiUrl}?wallet_address=${encodeURIComponent(walletAddress)}`
      );
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();

      if (data.balance_data) {
        // Update the balance display
        if (typeof updateBalance === "function") {
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
}

// Global instances
let walletManager;
let transactionManager;

// Global functions for button handlers
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

async function signTransaction() {
  const statusElement = document.getElementById("transaction-status");
  const statusMessage = document.getElementById("status-message");

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

    // Execute the transaction with progress feedback
    const result = await executeTransactionWithFeedback();

    // If we have the new HTML structure (for deployment page)
    if (
      successState &&
      (transactionManager.sessionData.session_type === "deploy" ||
        transactionManager.sessionData.session_type === "deploy_uniswap")
    ) {
      // Hide the main content and show success state
      const content = document.getElementById("content");
      if (content) content.style.display = "none";

      // Handle different deployment types
      if (transactionManager.sessionData.session_type === "deploy_uniswap") {
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
      } else {
        // Regular deployment - single contract
        if (contractAddressDisplay && result.contractAddress) {
          contractAddressDisplay.textContent = result.contractAddress;
        } else if (contractAddressDisplay) {
          // Check if the showContractAddressUnavailable function exists (from deploy.html)
          if (typeof showContractAddressUnavailable === "function") {
            showContractAddressUnavailable();
          } else {
            contractAddressDisplay.textContent =
              "Contract address not available";
            contractAddressDisplay.parentElement.classList.add("opacity-50");
          }
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
      let message = `✅ Transaction confirmed!<br>Hash: <code>${
        result.txHash || result
      }</code>`;

      // If it's a deployment and we have a contract address, show it
      if (
        result.contractAddress &&
        transactionManager.sessionData.session_type === "deploy"
      ) {
        message += `<br>📋 Contract Address: <code>${result.contractAddress}</code>`;
      } else if (transactionManager.sessionData.session_type === "deploy") {
        message += `<br>⚠️ Contract address not available - check blockchain explorer`;
      }

      if (statusMessage) statusMessage.innerHTML = message;
      if (statusElement) {
        statusElement.className =
          "bg-green-50 border border-green-200 text-green-800 px-6 py-4 rounded-xl shadow-sm";
      }
    }

    // Hide the sign button
    const signButton = document.getElementById("sign-button");
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

async function executeTransactionWithFeedback() {
  const statusMessage = document.getElementById("status-message");

  try {
    transactionManager.waitForContractAddress = async function (
      txHash,
      maxAttempts = 30
    ) {
      if (statusMessage) {
        statusMessage.innerHTML = `
          <div class="flex items-center justify-center">
            <div class="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-500 mr-3"></div>
            Transaction submitted! Waiting for confirmation...
          </div>
        `;
      }

      for (let attempt = 1; attempt <= maxAttempts; attempt++) {
        if (statusMessage && attempt > 1) {
          const elapsed = attempt * 2;
          statusMessage.innerHTML = `
            <div class="flex items-center justify-center">
              <div class="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-500 mr-3"></div>
              Mining transaction... (${elapsed}s elapsed)
            </div>
          `;
        }

        try {
          const receipt = await this.walletManager.selectedWallet.request({
            method: "eth_getTransactionReceipt",
            params: [txHash],
          });

          if (receipt) {
            if (receipt.contractAddress) {
              console.log(`Contract address found: ${receipt.contractAddress}`);
              if (statusMessage) {
                statusMessage.innerHTML = `
                  <div class="flex items-center justify-center text-green-600">
                    <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                    </svg>
                    Transaction confirmed! Contract deployed.
                  </div>
                `;
              }
              return receipt.contractAddress;
            } else if (receipt.status === "0x0") {
              throw new Error("Transaction failed during execution");
            } else {
              console.warn(
                "Transaction succeeded but no contract address found"
              );
              if (statusMessage) {
                statusMessage.innerHTML = `
                  <div class="flex items-center justify-center text-yellow-600">
                    <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01"></path>
                    </svg>
                    Transaction confirmed, but contract address unavailable.
                  </div>
                `;
              }
              return null;
            }
          }

          if (attempt < maxAttempts) {
            console.log(
              `Waiting for transaction confirmation... (attempt ${attempt}/${maxAttempts})`
            );
            await new Promise((resolve) => setTimeout(resolve, 2000));
          }
        } catch (error) {
          if (attempt === maxAttempts) {
            throw new Error(
              `Failed to get transaction receipt after ${maxAttempts} attempts: ${error.message}`
            );
          }

          console.warn(`Attempt ${attempt} failed:`, error.message);
          await new Promise((resolve) => setTimeout(resolve, 2000));
        }
      }

      throw new Error(
        "Transaction confirmation timeout - please check the blockchain explorer"
      );
    };

    return await transactionManager.executeTransaction();
  } catch (error) {
    throw error;
  }
}

// Global function for balance query
async function fetchBalanceFromWallet() {
  try {
    if (!walletManager || !walletManager.account) {
      alert("Please connect your wallet first");
      return;
    }

    const walletAddress = walletManager.account;

    // Update button state
    const button = document.getElementById("sign-button");
    if (button) {
      const originalHTML = button.innerHTML;
      button.innerHTML =
        '<div class="flex items-center justify-center"><div class="animate-spin rounded-full h-5 w-5 border-b-2 border-white mr-3"></div>Fetching Balance...</div>';
      button.disabled = true;
    }

    // Fetch balance data
    await transactionManager.fetchBalanceData(walletAddress);
  } catch (error) {
    console.error("Failed to fetch balance:", error);
    alert("Failed to fetch balance: " + error.message);

    // Reset button state
    const button = document.getElementById("sign-button");
    if (button) {
      button.innerHTML = "Fetch Balance";
      button.disabled = false;
    }
  }
}

// Initialize on page load
document.addEventListener("DOMContentLoaded", function () {
  walletManager = new WalletManager();
  transactionManager = new TransactionManager(walletManager);

  // Load session data if available
  const sessionData = document.getElementById("session-data");
  if (sessionData) {
    const sessionId = sessionData.dataset.sessionId;
    const apiUrl = sessionData.dataset.apiUrl;

    if (sessionId && apiUrl) {
      transactionManager.loadSessionData(sessionId, apiUrl);
    }
  }
});
