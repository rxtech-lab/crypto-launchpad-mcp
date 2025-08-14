#!/bin/bash

set -e

echo "Running post-installation configuration for Crypto Launchpad MCP..."

# Ensure the binary is executable
chmod +x /usr/local/bin/launchpad-mcp

# Create a simple configuration helper
echo "Setting up Crypto Launchpad MCP server..."

# Display installation success message
echo "✓ Post-installation configuration completed successfully!"
echo
echo "🚀 Crypto Launchpad MCP Server has been installed!"
echo
echo "To use the server with Claude Desktop:"
echo "1. Add the following to your Claude Desktop MCP configuration:"
echo
echo "   {"
echo "     \"launchpad-mcp\": {"
echo "       \"command\": \"/usr/local/bin/launchpad-mcp\","
echo "       \"args\": []"
echo "     }"
echo "   }"
echo
echo "2. Restart Claude Desktop to load the new MCP server"
echo
echo "🔗 The server provides 14 tools for crypto launchpad operations:"
echo "   • Chain management (select-chain, set-chain)"
echo "   • Template management (list-template, create-template, update-template)"
echo "   • Token deployment (launch)"
echo "   • Uniswap integration (8 tools for liquidity and trading)"
echo
echo "📊 Database location: ~/launchpad.db"
echo "🌐 Web interface: http://localhost:[random-port] (assigned automatically)"
echo
echo "For more information, visit: https://github.com/rxtech-lab/launchpad-mcp"