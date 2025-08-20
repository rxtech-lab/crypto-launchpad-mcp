# Crypto Launchpad MCP Server

AI-powered crypto launchpad supporting Ethereum and Solana blockchains. Unlike traditional web-based launchpads, this uses AI as the interface for easy token creation and management.

## Download

Visit [cryptolaunch.app](https://cryptolaunch.app) to download the latest version of the Crypto Launchpad MCP server for your platform.

## Features

- **AI-First Interface**: Natural language commands for blockchain operations
- **Multi-Chain Support**: Ethereum and Solana blockchain integration
- **Smart Contract Templates**: Pre-built and customizable contract templates
- **Uniswap Integration**: Complete DEX functionality for liquidity management
- **Secure Signing**: EIP-6963 wallet discovery with client-side transaction signing
- **Real-time Monitoring**: Pool metrics and transaction tracking

## Quick Start

### Prerequisites

- Go 1.24.5 or later
- Modern web browser with wallet extension (MetaMask, Coinbase Wallet, etc.)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd launchpad-mcp
```

2. Install dependencies:
```bash
make deps
```

3. Build the project:
```bash
make build
```

4. Run the server:
```bash
make run
```

## MCP Tools

The server provides 14 MCP tools for comprehensive crypto launchpad operations:

### Chain Management
- `select-chain` - Select blockchain (ethereum/solana)
- `set-chain` - Configure RPC and chain ID

### Template Management
- `list-template` - Search contract templates
- `create-template` - Create new templates
- `update-template` - Modify existing templates

### Token Deployment
- `launch` - Deploy contracts with signing interface

### Uniswap Integration
- `set-uniswap-version` - Configure Uniswap version
- `create-liquidity-pool` - Create new pools
- `add-liquidity` - Add liquidity to pools
- `remove-liquidity` - Remove liquidity from pools
- `swap-tokens` - Execute token swaps
- `get-pool-info` - View pool metrics
- `get-swap-quote` - Get swap estimates
- `monitor-pool` - Real-time pool monitoring

## Architecture

### Dual Server Design
- **MCP Server**: Handles AI tool requests via stdio
- **HTTP Server**: Provides transaction signing interfaces on random port

### Database
- SQLite database stored at `~/launchpad.db`
- Automatic migrations and schema management
- Session-based transaction tracking

### Frontend
- HTMX + Tailwind CSS for reactive interfaces
- EIP-6963 wallet discovery for maximum compatibility
- Client-side transaction signing for security

## Usage Examples

### Basic Workflow

1. **Setup Chain**:
```
AI: Please select Ethereum as the active blockchain
Tool: select-chain(chain_type="ethereum")
```

2. **Configure Network**:
```
AI: Set up Sepolia testnet
Tool: set-chain(chain_type="ethereum", rpc="https://sepolia.infura.io/v3/...", chain_id="11155111")
```

3. **Deploy Token**:
```
AI: Deploy a token called "MyToken" with symbol "MTK"
Tool: launch(template_id="1", token_name="MyToken", token_symbol="MTK", deployer_address="0x...")
Result: Signing URL generated for wallet connection
```

4. **Create Liquidity Pool**:
```
AI: Create a Uniswap pool with 1000 tokens and 1 ETH
Tool: create-liquidity-pool(token_address="0x...", initial_token_amount="1000", initial_eth_amount="1")
Result: Pool creation URL for user signing
```

### Transaction Signing Flow

1. AI tool generates unique signing URL
2. User opens URL in browser
3. Frontend loads with transaction details
4. User connects wallet via EIP-6963
5. User reviews and signs transaction
6. Real-time status updates
7. Database updated with transaction hash

## Development

### Project Structure
```
├── cmd/main.go              # Application entry point
├── internal/
│   ├── api/                 # HTTP server for signing
│   ├── database/            # Database layer
│   ├── mcp/                 # MCP server
│   └── models/              # Data models
├── tools/                   # MCP tool implementations
├── templates/               # Frontend assets
└── docs/                    # Documentation
```

### Adding New Tools

1. Create tool file in `tools/` directory following the pattern:
```go
func NewMyTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
    tool := mcp.NewTool("my_tool", ...)
    handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Implementation
    }
    return tool, handler
}
```

2. Register tool in `internal/mcp/server.go`:
```go
myTool, myHandler := tools.NewMyTool(db)
srv.AddTool(myTool, myHandler)
```

### Testing

Run all tests:
```bash
make test
```

### Building

Build for production:
```bash
make build
```

Clean build artifacts:
```bash
make clean
```

## Security

- **No Private Keys**: System never handles private keys
- **Client-Side Signing**: All transactions signed in user's browser
- **Session Expiry**: 30-minute timeout for signing sessions
- **Input Validation**: All parameters validated before processing
- **Template Validation**: Smart contracts checked for security issues

## Supported Networks

### Ethereum
- Mainnet (Chain ID: 1)
- Sepolia (Chain ID: 11155111)
- Goerli (Chain ID: 5)
- Custom networks via RPC configuration

### Solana
- Mainnet Beta
- Devnet
- Testnet
- Custom RPC endpoints

## Limitations

- Uniswap v3/v4 support is experimental (v2 fully supported)
- Solana DEX integration not yet implemented
- Real-time price feeds require external APIs
- Advanced trading features (limit orders, etc.) not included

## Contributing

1. Fork the repository
2. Create feature branch
3. Add comprehensive tests
4. Follow existing code patterns
5. Submit pull request

## License

[License details here]

## Support

For issues and questions:
- GitHub Issues: [Repository issues page]
- Documentation: `docs/` directory
- Example code: `example/` directory