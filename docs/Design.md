# Crypto Launchpad MCP Server

## Overview

AI-powered crypto launchpad supporting Ethereum and Solana blockchains. Unlike traditional web-based launchpads, this uses AI as the interface for easy token creation and management.

## Architecture

### Tech Stack
- **Backend**: Go with Fiber HTTP framework
- **MCP Server**: go-mcp for AI tool integration
- **Database**: GORM with SQLite
- **Frontend**: HTMX + Tailwind CSS
- **Templates**: Go template engine with embedded templates from /templates folder
- **Wallet**: EIP-6963 for wallet discovery

### Core Components
- MCP Server for AI tools
- HTTP Server for web interface
- Database layer with GORM
- Template management system
- Blockchain integration layer

## MCP Tools

### Core Token Tools
- `select-chain`: Select blockchain (ethereum/solana). Stores selection in database
- `list-template(chainType, keyword, limit)`: List predefined templates with SQLite search
- `create-template(newTemplate, description, chainType)`: Create new template with syntax validation
- `update-template(templateId, description, chainType, newTemplate)`: Update existing template
- `set-chain(rpc, chainId, chainType)`: Configure target blockchain
- `launch(template)`: Generate deployment URL with contract compilation and signing interface

### Uniswap Integration Tools
- `set-uniswap-version(version)`: Set Uniswap version (v2/v3/v4, currently only v2 supported). Stores version in database
- `create-liquidity-pool(tokenAddress, initialTokenAmount, initialETHAmount)`: Create new Uniswap liquidity pool with signing interface
- `add-liquidity(tokenAddress, tokenAmount, ethAmount, minTokenAmount, minETHAmount)`: Add liquidity to existing pool with signing interface
- `remove-liquidity(tokenAddress, liquidityAmount, minTokenAmount, minETHAmount)`: Remove liquidity from pool with signing interface
- `swap-tokens(fromToken, toToken, amount, slippageTolerance)`: Execute token swaps via Uniswap with signing interface
- `get-pool-info(tokenAddress)`: Retrieve pool metrics (reserves, liquidity, price, volume) - read-only
- `get-swap-quote(fromToken, toToken, amount)`: Get swap estimates and price impact - read-only
- `monitor-pool(tokenAddress)`: Real-time pool monitoring and event tracking - read-only

## Database Schema

### Core Tables
- `chains`: User-selected blockchain configurations
- `templates`: Smart contract templates by chain type
- `deployments`: Deployed token contracts
- `uniswap_settings`: Uniswap version and configuration

### Uniswap Tables
- `liquidity_pools`: Created pool information
- `liquidity_positions`: User liquidity positions
- `swap_transactions`: Historical swap data

## Transaction Signing Interface

For operations requiring user signatures, the system generates a frontend page with:

### UI Components
- **Transaction Details**: Clear display of operation type, amounts, and recipient
- **Connect Wallet Button**: EIP-6963 wallet discovery and connection
- **Network Switch**: Automatic network addition/switching if user on wrong chain
- **Sign & Send Button**: Execute transaction after user review
- **Transaction Status**: Real-time status updates and transaction hash display

### Signing Flow
1. Tool generates unique URL for transaction
2. Frontend loads with pre-populated transaction data
3. User connects wallet via EIP-6963
4. System checks/switches to correct network
5. User reviews transaction details
6. User signs and sends transaction
7. User signs ownership verification message (for security)
8. Real-time status updates until confirmation

### Security Enhancement - Transaction Ownership Verification

To prevent unauthorized transaction confirmations, the system implements signature-based ownership verification:

#### Process
- After transaction execution, user signs a timestamped message: "I am signing into Launchpad at {timestamp}"
- Backend verifies the signature matches the transaction sender address
- Only verified ownership allows transaction status updates

#### API Integration
- `POST /api/tx/{sessionId}/transaction/{index}` now includes optional `signature` field
- Backend calls `utils.TransactionOwnershipBySignature()` to verify ownership
- Returns 401 Unauthorized if verification fails

## Workflow

1. **Setup**: User selects chain and Uniswap version via AI tools
2. **Template**: AI lists/creates appropriate contract templates
3. **Deploy**: Launch tool compiles and generates signing interface for deployment
4. **Uniswap**: AI tools create pools, manage liquidity, execute swaps with signing interfaces
5. **Monitor**: Real-time tracking of pool performance and transactions