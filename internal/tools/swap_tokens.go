package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

func NewSwapTokensTool(chainService services.ChainService, liquidityService services.LiquidityService, uniswapService services.UniswapService, txService services.TransactionService, serverPort int) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("swap_tokens",
		mcp.WithDescription("Execute token swaps via Uniswap with signing interface. Generates a URL where users can connect wallet and sign the swap transaction."),
		mcp.WithString("from_token",
			mcp.Required(),
			mcp.Description(fmt.Sprintf("Address of the token to swap from (use %s for ETH)", services.ETH_TOKEN_ADDRESS)),
		),
		mcp.WithString("to_token",
			mcp.Required(),
			mcp.Description(fmt.Sprintf("Address of the token to swap to (use %s for ETH)", services.ETH_TOKEN_ADDRESS)),
		),
		mcp.WithString("amount",
			mcp.Required(),
			mcp.Description("Amount of tokens to swap"),
		),
		mcp.WithString("slippage_tolerance",
			mcp.Required(),
			mcp.Description("Maximum slippage tolerance as percentage (e.g., '0.5' for 0.5%)"),
		),
		mcp.WithString("user_address",
			mcp.Required(),
			mcp.Description("Address that will execute the swap"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	}

	return tool, handler
}

// swapEthToToken swaps ETH to a specified token
func swapEthToToken(toToken, amount, receiver string) (models.TransactionDeployment, error) {

}

func swapTokenToEth(fromToken, amount, receiver string) (models.TransactionDeployment, error) {

}

func swapTokenToToken(fromToken, toToken, amount, receiver string) (models.TransactionDeployment, error) {

}
