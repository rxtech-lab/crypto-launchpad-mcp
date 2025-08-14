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
        window.addEventListener('eip6963:announceProvider', (event) => {
            const { info, provider } = event.detail;
            this.wallets.set(info.uuid, { info, provider });
            this.updateWalletList();
        });

        // Request wallet announcements
        window.dispatchEvent(new Event('eip6963:requestProvider'));
    }

    updateWalletList() {
        const walletSelect = document.getElementById('wallet-select');
        if (!walletSelect) return;

        walletSelect.innerHTML = '<option value="">Select a wallet...</option>';
        
        for (const [uuid, wallet] of this.wallets) {
            const option = document.createElement('option');
            option.value = uuid;
            option.textContent = wallet.info.name;
            walletSelect.appendChild(option);
        }
    }

    async connectWallet(walletUuid) {
        if (!this.wallets.has(walletUuid)) {
            throw new Error('Wallet not found');
        }

        const wallet = this.wallets.get(walletUuid);
        this.selectedWallet = wallet.provider;

        try {
            // Request account access
            const accounts = await this.selectedWallet.request({
                method: 'eth_requestAccounts'
            });

            if (accounts.length === 0) {
                throw new Error('No accounts available');
            }

            this.account = accounts[0];

            // Get current chain ID
            this.chainId = await this.selectedWallet.request({
                method: 'eth_chainId'
            });

            this.updateConnectionStatus();
            return { account: this.account, chainId: this.chainId };
        } catch (error) {
            console.error('Failed to connect wallet:', error);
            throw error;
        }
    }

    async switchNetwork(targetChainId) {
        if (!this.selectedWallet) {
            throw new Error('No wallet connected');
        }

        const chainIdHex = '0x' + parseInt(targetChainId).toString(16);

        try {
            await this.selectedWallet.request({
                method: 'wallet_switchEthereumChain',
                params: [{ chainId: chainIdHex }]
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
            '1': {
                chainId: '0x1',
                chainName: 'Ethereum Mainnet',
                rpcUrls: ['https://ethereum.publicnode.com'],
                nativeCurrency: { name: 'ETH', symbol: 'ETH', decimals: 18 },
                blockExplorerUrls: ['https://etherscan.io']
            },
            '11155111': {
                chainId: '0xaa36a7',
                chainName: 'Sepolia',
                rpcUrls: ['https://sepolia.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161'],
                nativeCurrency: { name: 'SepoliaETH', symbol: 'ETH', decimals: 18 },
                blockExplorerUrls: ['https://sepolia.etherscan.io']
            }
        };

        const networkConfig = networks[chainId];
        if (!networkConfig) {
            throw new Error('Network configuration not found');
        }

        await this.selectedWallet.request({
            method: 'wallet_addEthereumChain',
            params: [networkConfig]
        });

        this.chainId = networkConfig.chainId;
        this.updateConnectionStatus();
    }

    async signTransaction(transactionData) {
        if (!this.selectedWallet || !this.account) {
            throw new Error('Wallet not connected');
        }

        try {
            const txHash = await this.selectedWallet.request({
                method: 'eth_sendTransaction',
                params: [transactionData]
            });

            return txHash;
        } catch (error) {
            console.error('Transaction failed:', error);
            throw error;
        }
    }

    updateConnectionStatus() {
        const statusElement = document.getElementById('connection-status');
        const connectButton = document.getElementById('connect-button');
        const signButton = document.getElementById('sign-button');

        if (this.account) {
            if (statusElement) {
                statusElement.innerHTML = `
                    <div class="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded">
                        <p><strong>Connected:</strong> ${this.account.slice(0, 6)}...${this.account.slice(-4)}</p>
                        <p><strong>Chain ID:</strong> ${parseInt(this.chainId, 16)}</p>
                    </div>
                `;
            }
            if (connectButton) connectButton.style.display = 'none';
            if (signButton) signButton.style.display = 'block';
        } else {
            if (statusElement) {
                statusElement.innerHTML = `
                    <div class="bg-yellow-100 border border-yellow-400 text-yellow-700 px-4 py-3 rounded">
                        <p>Please connect your wallet to continue</p>
                    </div>
                `;
            }
            if (connectButton) connectButton.style.display = 'block';
            if (signButton) signButton.style.display = 'none';
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
                throw new Error('Failed to load session data');
            }
            this.sessionData = await response.json();
            this.displayTransactionDetails();
        } catch (error) {
            console.error('Error loading session data:', error);
            this.displayError('Failed to load transaction details');
        }
    }

    displayTransactionDetails() {
        const contentElement = document.getElementById('content');
        if (!contentElement || !this.sessionData) return;

        const { transaction_data, session_type, chain_type, chain_id } = this.sessionData;

        let detailsHTML = '';
        
        switch (session_type) {
            case 'deploy':
                detailsHTML = this.generateDeploymentDetails(transaction_data);
                break;
            case 'create_pool':
                detailsHTML = this.generateCreatePoolDetails(transaction_data);
                break;
            case 'add_liquidity':
                detailsHTML = this.generateAddLiquidityDetails(transaction_data);
                break;
            case 'remove_liquidity':
                detailsHTML = this.generateRemoveLiquidityDetails(transaction_data);
                break;
            case 'swap':
                detailsHTML = this.generateSwapDetails(transaction_data);
                break;
            default:
                detailsHTML = '<p class="text-red-600">Unknown transaction type</p>';
        }

        contentElement.innerHTML = `
            <div class="space-y-6">
                <div class="bg-blue-50 border border-blue-200 rounded-lg p-4">
                    <h2 class="text-lg font-semibold text-blue-800 mb-2">Transaction Details</h2>
                    ${detailsHTML}
                </div>

                <div class="bg-gray-50 border border-gray-200 rounded-lg p-4">
                    <h3 class="text-md font-semibold text-gray-800 mb-2">Network Information</h3>
                    <p><strong>Chain Type:</strong> ${chain_type}</p>
                    <p><strong>Chain ID:</strong> ${chain_id}</p>
                </div>

                <div id="connection-status"></div>

                <div class="space-y-4">
                    <div id="connect-button">
                        <select id="wallet-select" class="w-full p-3 border border-gray-300 rounded-lg">
                            <option value="">Select a wallet...</option>
                        </select>
                        <button onclick="connectWallet()" class="mt-2 w-full bg-blue-600 text-white py-3 px-6 rounded-lg hover:bg-blue-700 transition-colors">
                            Connect Wallet
                        </button>
                    </div>

                    <button id="sign-button" onclick="signTransaction()" style="display: none;" class="w-full bg-green-600 text-white py-3 px-6 rounded-lg hover:bg-green-700 transition-colors">
                        Sign & Send Transaction
                    </button>
                </div>

                <div id="transaction-status" class="hidden">
                    <div class="bg-yellow-100 border border-yellow-400 text-yellow-700 px-4 py-3 rounded">
                        <p id="status-message">Processing transaction...</p>
                    </div>
                </div>
            </div>
        `;

        this.walletManager.updateConnectionStatus();
    }

    generateDeploymentDetails(data) {
        return `
            <div class="space-y-2">
                <p><strong>Token Name:</strong> ${data.token_name}</p>
                <p><strong>Token Symbol:</strong> ${data.token_symbol}</p>
                <p><strong>Deployer Address:</strong> ${data.deployer_address}</p>
                <p><strong>Template:</strong> Smart Contract Template</p>
            </div>
        `;
    }

    generateCreatePoolDetails(data) {
        return `
            <div class="space-y-2">
                <p><strong>Token Address:</strong> ${data.token_address}</p>
                <p><strong>Initial Token Amount:</strong> ${data.initial_token_amount}</p>
                <p><strong>Initial ETH Amount:</strong> ${data.initial_eth_amount}</p>
                <p><strong>Uniswap Version:</strong> ${data.uniswap_version}</p>
                <p><strong>Creator Address:</strong> ${data.creator_address}</p>
            </div>
        `;
    }

    generateAddLiquidityDetails(data) {
        return `
            <div class="space-y-2">
                <p><strong>Token Address:</strong> ${data.token_address}</p>
                <p><strong>Token Amount:</strong> ${data.token_amount}</p>
                <p><strong>ETH Amount:</strong> ${data.eth_amount}</p>
                <p><strong>Min Token Amount:</strong> ${data.min_token_amount}</p>
                <p><strong>Min ETH Amount:</strong> ${data.min_eth_amount}</p>
                <p><strong>User Address:</strong> ${data.user_address}</p>
            </div>
        `;
    }

    generateRemoveLiquidityDetails(data) {
        return `
            <div class="space-y-2">
                <p><strong>Token Address:</strong> ${data.token_address}</p>
                <p><strong>Liquidity Amount:</strong> ${data.liquidity_amount}</p>
                <p><strong>Min Token Amount:</strong> ${data.min_token_amount}</p>
                <p><strong>Min ETH Amount:</strong> ${data.min_eth_amount}</p>
                <p><strong>User Address:</strong> ${data.user_address}</p>
            </div>
        `;
    }

    generateSwapDetails(data) {
        const fromDisplay = data.from_token === '0x0' ? 'ETH' : data.from_token;
        const toDisplay = data.to_token === '0x0' ? 'ETH' : data.to_token;
        
        return `
            <div class="space-y-2">
                <p><strong>From Token:</strong> ${fromDisplay}</p>
                <p><strong>To Token:</strong> ${toDisplay}</p>
                <p><strong>Amount:</strong> ${data.amount}</p>
                <p><strong>Slippage Tolerance:</strong> ${data.slippage_tolerance}%</p>
                <p><strong>User Address:</strong> ${data.user_address}</p>
            </div>
        `;
    }

    displayError(message) {
        const contentElement = document.getElementById('content');
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
            throw new Error('Wallet not connected');
        }

        if (!this.sessionData) {
            throw new Error('No transaction data loaded');
        }

        // Check if we're on the correct network
        const targetChainId = this.sessionData.chain_id;
        const currentChainId = parseInt(this.walletManager.getChainId(), 16).toString();
        
        if (currentChainId !== targetChainId) {
            await this.walletManager.switchNetwork(targetChainId);
        }

        // Prepare transaction based on type
        const transactionData = this.prepareTransactionData();
        
        // Sign and send transaction
        const txHash = await this.walletManager.signTransaction(transactionData);
        
        // Update session status
        await this.updateSessionStatus(txHash, 'confirmed');
        
        return txHash;
    }

    prepareTransactionData() {
        const { transaction_data } = this.sessionData;
        
        // This is a simplified transaction preparation
        // In a real implementation, this would use proper contract ABIs and encoding
        return {
            from: this.walletManager.getAccount(),
            to: transaction_data.contract_address || transaction_data.token_address,
            value: transaction_data.eth_amount ? `0x${parseInt(transaction_data.eth_amount).toString(16)}` : '0x0',
            gas: '0x186a0', // 100000 gas limit
            gasPrice: '0x9184e72a000', // 10 gwei
            data: '0x' // Contract data would be encoded here
        };
    }

    async updateSessionStatus(txHash, status) {
        const confirmUrl = window.location.pathname.replace(/\/([^\/]+)$/, '/confirm');
        
        try {
            const response = await fetch(`/api${confirmUrl}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    transaction_hash: txHash,
                    status: status
                })
            });

            if (!response.ok) {
                throw new Error('Failed to update session status');
            }
        } catch (error) {
            console.error('Error updating session status:', error);
        }
    }
}

// Global instances
let walletManager;
let transactionManager;

// Global functions for button handlers
async function connectWallet() {
    const walletSelect = document.getElementById('wallet-select');
    const selectedWallet = walletSelect.value;
    
    if (!selectedWallet) {
        alert('Please select a wallet');
        return;
    }

    try {
        await walletManager.connectWallet(selectedWallet);
    } catch (error) {
        alert('Failed to connect wallet: ' + error.message);
    }
}

async function signTransaction() {
    const statusElement = document.getElementById('transaction-status');
    const statusMessage = document.getElementById('status-message');
    
    try {
        statusElement.classList.remove('hidden');
        statusMessage.textContent = 'Processing transaction...';
        
        const txHash = await transactionManager.executeTransaction();
        
        statusMessage.textContent = `Transaction submitted! Hash: ${txHash}`;
        statusElement.className = 'bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded';
        
        // Hide the sign button
        document.getElementById('sign-button').style.display = 'none';
        
    } catch (error) {
        statusMessage.textContent = `Transaction failed: ${error.message}`;
        statusElement.className = 'bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded';
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    walletManager = new WalletManager();
    transactionManager = new TransactionManager(walletManager);
    
    // Load session data if available
    const sessionData = document.getElementById('session-data');
    if (sessionData) {
        const sessionId = sessionData.dataset.sessionId;
        const apiUrl = sessionData.dataset.apiUrl;
        
        if (sessionId && apiUrl) {
            transactionManager.loadSessionData(sessionId, apiUrl);
        }
    }
});