package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type setUniswapAddressesTool struct {
	uniswapService services.UniswapService
	chainService   services.ChainService
}

type SetUniswapAddressesArguments struct {
	// Required fields
	Version string `json:"version" validate:"required"`

	// Optional address fields - at least one must be provided
	FactoryAddress *string `json:"factory_address,omitempty"`
	RouterAddress  *string `json:"router_address,omitempty"`
	WETHAddress    *string `json:"weth_address,omitempty"`
}

func NewSetUniswapAddressesTool(uniswapService services.UniswapService, chainService services.ChainService) *setUniswapAddressesTool {
	return &setUniswapAddressesTool{
		uniswapService: uniswapService,
		chainService:   chainService,
	}
}

func (s *setUniswapAddressesTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("set_uniswap_addresses",
		mcp.WithDescription("Set or update Uniswap contract addresses (factory, router, WETH) for cases where contracts were deployed externally. Creates a new deployment record if none exists for the active chain. At least one address must be provided."),
		mcp.WithString("version",
			mcp.Required(),
			mcp.Description("Uniswap version (v2, v3, or v4)"),
		),
		mcp.WithString("factory_address",
			mcp.Description("Factory contract address (0x prefixed hex string)"),
		),
		mcp.WithString("router_address",
			mcp.Description("Router contract address (0x prefixed hex string)"),
		),
		mcp.WithString("weth_address",
			mcp.Description("WETH contract address (0x prefixed hex string)"),
		),
	)

	return tool
}

func (s *setUniswapAddressesTool) GetHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args SetUniswapAddressesArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		// Validate that at least one address is provided
		if args.FactoryAddress == nil && args.RouterAddress == nil && args.WETHAddress == nil {
			return mcp.NewToolResultError("At least one address (factory_address, router_address, or weth_address) must be provided"), nil
		}

		// Validate version
		if err := utils.ValidateUniswapVersion(args.Version); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Validate address formats
		if args.FactoryAddress != nil {
			if err := validateEthereumAddress(*args.FactoryAddress); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid factory_address: %v", err)), nil
			}
		}
		if args.RouterAddress != nil {
			if err := validateEthereumAddress(*args.RouterAddress); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid router_address: %v", err)), nil
			}
		}
		if args.WETHAddress != nil {
			if err := validateEthereumAddress(*args.WETHAddress); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid weth_address: %v", err)), nil
			}
		}

		user, _ := utils.GetAuthenticatedUser(ctx)
		var userId *string
		if user != nil {
			userId = &user.Sub
		}

		// Get active chain configuration
		activeChain, err := s.chainService.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Validate that only Ethereum is supported
		if activeChain.ChainType != models.TransactionChainTypeEthereum {
			return mcp.NewToolResultError(fmt.Sprintf("Uniswap is only supported on Ethereum, got %s", activeChain.ChainType)), nil
		}

		// Get or create Uniswap deployment record
		existingDeployment, err := s.uniswapService.GetUniswapDeploymentByChain(activeChain.ID)
		var deploymentID uint

		if err != nil || existingDeployment == nil {
			// Create new deployment record
			deploymentID, err = s.uniswapService.CreateUniswapDeployment(activeChain.ID, args.Version, userId)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to create Uniswap deployment record: %v", err)), nil
			}
		} else {
			deploymentID = existingDeployment.ID
		}

		// Update addresses
		var updatedFields []string
		if args.FactoryAddress != nil {
			if err := s.uniswapService.UpdateFactoryAddress(deploymentID, *args.FactoryAddress); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to update factory address: %v", err)), nil
			}
			updatedFields = append(updatedFields, "factory_address")
		}
		if args.RouterAddress != nil {
			if err := s.uniswapService.UpdateRouterAddress(deploymentID, *args.RouterAddress); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to update router address: %v", err)), nil
			}
			updatedFields = append(updatedFields, "router_address")
		}
		if args.WETHAddress != nil {
			if err := s.uniswapService.UpdateWETHAddress(deploymentID, *args.WETHAddress); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to update WETH address: %v", err)), nil
			}
			updatedFields = append(updatedFields, "weth_address")
		}

		// Fetch updated deployment to return
		updatedDeployment, err := s.uniswapService.GetUniswapDeployment(deploymentID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve updated deployment: %v", err)), nil
		}

		result := map[string]interface{}{
			"id":              updatedDeployment.ID,
			"version":         updatedDeployment.Version,
			"chain_id":        updatedDeployment.ChainID,
			"factory_address": updatedDeployment.FactoryAddress,
			"router_address":  updatedDeployment.RouterAddress,
			"weth_address":    updatedDeployment.WETHAddress,
			"status":          updatedDeployment.Status,
			"updated_fields":  updatedFields,
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Successfully updated Uniswap %s addresses for chain %s", args.Version, activeChain.Name)),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}
}

// validateEthereumAddress validates that the address is a valid Ethereum address format
func validateEthereumAddress(address string) error {
	// Remove 0x prefix if present
	addr := strings.TrimPrefix(address, "0x")

	// Check length (should be 40 hex characters)
	if len(addr) != 40 {
		return fmt.Errorf("address must be 40 hex characters (excluding 0x prefix), got %d", len(addr))
	}

	// Check if all characters are valid hex
	for _, c := range addr {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("address contains invalid hex character: %c", c)
		}
	}

	return nil
}
