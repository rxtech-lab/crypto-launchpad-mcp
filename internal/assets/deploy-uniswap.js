// Uniswap Deployment Management
class UniswapDeploymentManager {
  constructor(walletManager) {
    this.walletManager = walletManager;
    this.sessionData = null;
    this.apiUrl = null;
    this.contractData = null;
  }

  async loadSessionData(sessionId, apiUrl) {
    this.apiUrl = apiUrl;
    try {
      const response = await fetch(apiUrl);
      if (!response.ok) {
        throw new Error("Failed to load session data");
      }
      this.sessionData = await response.json();
      this.contractData = this.sessionData.contract_data || {};
      this.displayTransactionDetails();
    } catch (error) {
      console.error("Error loading session data:", error);
      this.displayError("Failed to load transaction details");
    }
  }

  displayTransactionDetails() {
    const contentElement = document.getElementById("content");
    const deploymentInfoElement = document.getElementById("deployment-info");

    if (!contentElement || !this.sessionData) return;

    const { deployment_data, session_type, metadata } = this.sessionData;

    if (session_type !== "deploy_uniswap") {
      console.error(
        "Invalid session type for Uniswap deployment:",
        session_type
      );
      this.displayError("Invalid session type");
      return;
    }

    // Show deployment info (replace loading state)
    if (deploymentInfoElement) {
      deploymentInfoElement.classList.remove("hidden");

      // Render metadata if available
      if (metadata && window.renderMetadata) {
        // Ensure metadata is an array
        const metadataArray = Array.isArray(metadata) ? metadata : [];
        window.renderMetadata(metadataArray);
      }
    }

    // Update wallet connection status
    this.walletManager.updateConnectionStatus();
  }

  generateUniswapDeploymentDetails(data) {
    const deploymentData = data.deployment_data;
    const metadata = data.metadata || [];

    let metadataHTML = "";
    // Ensure metadata is an array before using forEach
    const metadataArray = Array.isArray(metadata) ? metadata : [];
    metadataArray.forEach((item) => {
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

    try {
      // Real Uniswap contract deployment
      const result = {
        success: true,
        contractAddresses: {},
        transactionHashes: {},
      };

      // Update deployment progress UI
      this.updateDeploymentProgress("weth", "pending");

      // 1. Deploy WETH9 contract
      console.log("Deploying WETH9 contract...");
      const wethDeployment = await this.deployContract("WETH9");
      result.contractAddresses.weth = wethDeployment.contractAddress;
      result.transactionHashes.weth = wethDeployment.txHash;
      this.updateDeploymentProgress("weth", "success");

      // 2. Deploy Factory contract
      this.updateDeploymentProgress("factory", "pending");
      console.log("Deploying Factory contract...");
      const factoryDeployment = await this.deployContract("Factory", [
        this.walletManager.getAccount(),
      ]);
      result.contractAddresses.factory = factoryDeployment.contractAddress;
      result.transactionHashes.factory = factoryDeployment.txHash;
      this.updateDeploymentProgress("factory", "success");

      // 3. Deploy Router contract
      this.updateDeploymentProgress("router", "pending");
      console.log("Deploying Router contract...");
      const routerDeployment = await this.deployContract("Router", [
        factoryDeployment.contractAddress,
        wethDeployment.contractAddress,
      ]);
      result.contractAddresses.router = routerDeployment.contractAddress;
      result.transactionHashes.router = routerDeployment.txHash;
      this.updateDeploymentProgress("router", "success");

      console.log("All contracts deployed successfully:", result);

      // Update session with deployment results
      await this.updateUniswapSessionStatus(result);

      return result;
    } catch (error) {
      console.error("Deployment failed:", error);
      throw new Error(`Deployment failed: ${error.message}`);
    }
  }

  async deployContract(contractName, constructorArgs = []) {
    try {
      // Get contract bytecode and ABI from server
      const contractData = await this.getContractData(contractName);

      // Prepare transaction data
      let data = contractData.bytecode;
      if (constructorArgs.length > 0) {
        // For simplicity, we'll just append encoded constructor args
        // In a real implementation, you'd use proper ABI encoding
        const encodedArgs = this.encodeConstructorArgs(constructorArgs);
        data += encodedArgs;
      }

      // Ensure the final data hex string has even length
      // The data includes "0x" prefix, so we check the hex part
      const hexPart = data.slice(2);
      if (hexPart.length % 2 === 1) {
        data += "0";
      }

      const transactionData = {
        data: data,
        value: "0x0",
        gas: "0x7A1200", // 8,000,000 gas (increased for complex contracts)
      };

      console.log(`Deploying ${contractName} with transaction data:`, {
        contractName,
        dataLength: data.length,
        data: data.slice(0, 100) + "...", // Log first 100 chars
        gas: transactionData.gas,
        constructorArgs,
      });

      // Sign and send transaction using wallet
      const txHash = await this.walletManager.signTransaction(transactionData);
      console.log(`${contractName} deployment transaction sent: ${txHash}`);

      // Wait for transaction receipt to get contract address
      const contractAddress = await this.waitForContractAddress(txHash);
      console.log(`${contractName} deployed at address: ${contractAddress}`);

      return {
        txHash,
        contractAddress,
      };
    } catch (error) {
      console.error(`Failed to deploy ${contractName}:`, error);
      throw error;
    }
  }

  async getContractData(contractName) {
    try {
      // Use cached contract data from session response
      if (this.contractData && this.contractData[contractName]) {
        const contractInfo = this.contractData[contractName];
        return {
          bytecode: contractInfo.bytecode,
          abi: contractInfo.abi,
        };
      }

      // Fallback to server fetch if not cached
      const response = await fetch(`/api/contracts/${contractName}`);
      if (!response.ok) {
        throw new Error(
          `Failed to fetch contract ${contractName}: ${response.status}`
        );
      }
      const artifact = await response.json();

      // Ensure bytecode has 0x prefix
      let bytecode = artifact.bytecode || '';
      if (typeof bytecode === 'string' && !bytecode.startsWith("0x")) {
        bytecode = "0x" + bytecode;
      } else if (typeof bytecode !== 'string') {
        bytecode = '';
      }

      return {
        bytecode: bytecode,
        abi: artifact.abi,
      };
    } catch (error) {
      console.error(`Error getting contract ${contractName}:`, error);
      throw error; // Don't fallback, fail properly if contracts can't be fetched
    }
  }

  encodeConstructorArgs(args) {
    // Simplified constructor argument encoding
    // In a real implementation, you'd use proper ABI encoding
    let encoded = "";
    for (const arg of args) {
      if (typeof arg === "string" && arg.startsWith("0x")) {
        // Address - pad to 32 bytes
        encoded += arg.slice(2).padStart(64, "0");
      } else {
        // For simplicity, treat everything else as uint256
        const hex = parseInt(arg).toString(16);
        encoded += hex.padStart(64, "0");
      }
    }

    // Ensure the encoded arguments have even length
    if (encoded.length % 2 === 1) {
      encoded += "0";
    }

    return encoded;
  }

  async waitForContractAddress(txHash) {
    // Wait for transaction receipt to get contract address
    let retries = 30;
    while (retries > 0) {
      try {
        // Get the provider from walletManager
        const provider =
          this.walletManager.selectedWallet?.provider ||
          this.walletManager.selectedProvider ||
          window.ethereum;

        if (!provider) {
          throw new Error("No wallet provider available");
        }

        const receipt = await provider.request({
          method: "eth_getTransactionReceipt",
          params: [txHash],
        });

        console.log(`Receipt for ${txHash}:`, receipt);

        if (receipt && receipt.contractAddress) {
          console.log(
            `Contract deployed successfully at: ${receipt.contractAddress}`
          );
          return receipt.contractAddress;
        }

        if (receipt && receipt.status === "0x0") {
          console.error("Transaction failed with receipt:", receipt);
          throw new Error(
            `Transaction failed: ${txHash}. Status: ${receipt.status}, Gas used: ${receipt.gasUsed}`
          );
        }

        if (receipt === null) {
          console.log(
            `Transaction ${txHash} not yet mined, retrying... (${retries} retries left)`
          );
        }

        // Wait 2 seconds before retry
        await new Promise((resolve) => setTimeout(resolve, 2000));
        retries--;
      } catch (error) {
        console.error("Error waiting for contract address:", error);
        retries--;
        await new Promise((resolve) => setTimeout(resolve, 2000));
      }
    }

    throw new Error("Timeout waiting for contract deployment");
  }

  updateDeploymentProgress(contractName, status) {
    // Use the global updateProgress function from the HTML template
    if (window.updateProgress) {
      window.updateProgress(contractName, status);
    }
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
  const deploymentInfoElement = document.getElementById("deployment-info");
  const successState = document.getElementById("success-state");

  try {
    console.log("Starting Uniswap deployment...");

    // Execute the Uniswap deployment
    const result = await uniswapDeploymentManager.executeUniswapDeployment();

    console.log("Deployment completed successfully:", result);

    // Hide deployment info and loading content, show success state
    const loadingState = document.getElementById("loading-state");
    if (loadingState) {
      loadingState.style.display = "none";
    }
    if (deploymentInfoElement) {
      deploymentInfoElement.style.display = "none";
    }

    if (successState) {
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
    }
  } catch (error) {
    console.error("Deployment failed:", error);

    // Update deployment progress to show error
    if (window.updateProgress) {
      window.updateProgress("weth", "error");
      window.updateProgress("factory", "error");
      window.updateProgress("router", "error");
    }

    // Could show error state here if needed
    alert(`Deployment failed: ${error.message}`);
  }
}

// Initialize on page load
document.addEventListener("DOMContentLoaded", function () {
  // Only initialize if walletManager exists (from wallet-connection.js)
  if (typeof walletManager !== "undefined") {
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
