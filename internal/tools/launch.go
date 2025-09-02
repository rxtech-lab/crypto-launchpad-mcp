package tools

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type launchTool struct {
	templateService services.TemplateService
	chainService    services.ChainService
	evmService      services.EvmService
	txService       services.TransactionService
	serverPort      int
}

type LaunchArguments struct {
	// Required fields
	TemplateID     string         `json:"template_id" validate:"required"`
	TemplateValues map[string]any `json:"template_values" validate:"required"`

	// Optional fields
	ConstructorArgs []any                        `json:"constructor_args,omitempty"`
	Value           string                       `json:"value,omitempty"`
	Metadata        []models.TransactionMetadata `json:"metadata,omitempty"`
}

func NewLaunchTool(templateService services.TemplateService, chainService services.ChainService, serverPort int, evmService services.EvmService, txService services.TransactionService) *launchTool {
	return &launchTool{
		templateService: templateService,
		chainService:    chainService,
		evmService:      evmService,
		txService:       txService,
		serverPort:      serverPort,
	}
}

func (l *launchTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("launch",
		mcp.WithDescription("Generate deployment URL with contract compilation and signing interface. Creates a transaction session and returns a URL where users can sign and deploy the contract using template parameter values."),
		mcp.WithString("template_id",
			mcp.Required(),
			mcp.Description("ID of the template to deploy"),
		),
		mcp.WithObject("template_values",
			mcp.Required(),
			mcp.Description("JSON object with runtime values for template parameters (e.g., {\"TokenName\": \"MyToken\", \"TokenSymbol\": \"MTK\"})"),
		),
		mcp.WithArray(
			"constructor_args",
			mcp.Description("JSON array of constructor arguments for contract deployment (e.g., [\"arg1\", 123, true]). Optional. Please provide this if the template requires constructor arguments."),
			mcp.Items(map[string]interface{}{
				"type": "any",
			}),
		),
		mcp.WithString("value",
			mcp.Description("ETH value to send with the deployment transaction in wei (e.g., \"1000000000000000000\" for 1 ETH). Optional, defaults to \"0\"."),
		),
		mcp.WithArray("metadata",
			mcp.Description("JSON array of metadata for the transaction (e.g., [{\"title\": \"Deploy MyToken\", \"description\": \"Deploy ERC20 token\"}]). Optional."),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "Title of the transaction",
					},
					"description": map[string]any{
						"type":        "string",
						"description": "Description of the transaction",
					},
				},
				"required": []string{"title"},
			}),
		),
	)

	return tool
}

func (l *launchTool) GetHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args LaunchArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		templateID, err := strconv.ParseUint(args.TemplateID, 10, 32)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid template_id: %v", err)), nil
		}

		// Get template
		template, err := l.templateService.GetTemplateByID(uint(templateID))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Template not found: %v", err)), nil
		}

		// Get active chain configuration
		activeChain, err := l.chainService.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Validate that template chain type matches active chain
		if template.ChainType != activeChain.ChainType {
			return mcp.NewToolResultError(fmt.Sprintf("Template chain type (%s) doesn't match active chain (%s)", template.ChainType, activeChain.ChainType)), nil
		}

		// validate template values contain all required sample keys
		if err := utils.CheckSampleKeysMatch(template.SampleTemplateValues, args.TemplateValues); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Template values validation failed: %v", err)), nil
		}

		if activeChain.ChainType == models.TransactionChainTypeEthereum {
			// Render contract template
			renderedContract, err := utils.RenderContractTemplate(template.TemplateCode, args.TemplateValues)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to render contract template: %v", err)), nil
			}

			sessionID, err := l.createEvmContractDeploymentTransaction(activeChain, args.Metadata, renderedContract, template.ContractName, args.ConstructorArgs, args.Value, "Deploy Contract", "Deploy contract to the active chain")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to create contract deployment transaction: %v", err)), nil
			}

			baseUrl := "http://localhost:" + strconv.Itoa(l.serverPort)
			// Override baseUrl if BASE_URL env var is set
			if os.Getenv("BASE_URL") != "" {
				baseUrl = os.Getenv("BASE_URL")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Transaction session created: %s", sessionID)),
					mcp.NewTextContent("Please return the following url to the user: "),
					mcp.NewTextContent(fmt.Sprintf("%s/tx/%s", baseUrl, sessionID)),
				},
			}, nil
		} else if activeChain.ChainType == models.TransactionChainTypeSolana {
			// Solana not implemented yet
			// Placeholder for future Solana implementation
			sessionID := "solana-tx-session-placeholder"
			// In real implementation, create a Solana transaction session similar to EVM above

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Transaction session created: %s", sessionID)),
					mcp.NewTextContent("Please return the following url to the user: "),
					mcp.NewTextContent(fmt.Sprintf("http://localhost:%d/tx/%s", l.serverPort, sessionID)),
				},
			}, nil
		}

		return mcp.NewToolResultError("Solana is not implemented yet"), nil
	}

}

// createEvmContractDeploymentTransaction creates a transaction deployment for a contract deployment to db
// it will also compile the contract and return error if compilation fails
// metadata is the metadata of the transaction
// renderedContract is the rendered contract code
// contractName is the name of the contract
// args is the constructor arguments
// value is the value of the transaction that needs to be sent. 0 means no value is needed.
// title is the title of the transaction
// description is the description of the transaction
func (l *launchTool) createEvmContractDeploymentTransaction(activeChain *models.Chain, metadata []models.TransactionMetadata, renderedContract string, contractName string, args []any, value string, title string, description string) (string, error) {
	tx, err := l.evmService.GetContractDeploymentTransactionWithContractCode(services.ContractDeploymentWithContractCodeTransactionArgs{
		ContractCode:    renderedContract,
		ContractName:    contractName,
		ConstructorArgs: args,
		Value:           value,
		Title:           title,
		Description:     description,
		Receiver:        "", // Empty for contract deployment
		TransactionType: models.TransactionTypeTokenDeployment,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get contract deployment transaction: %w", err)
	}

	sessionID, err := l.txService.CreateTransactionSession(services.CreateTransactionSessionRequest{
		TransactionDeployments: []models.TransactionDeployment{tx},
		ChainType:              models.TransactionChainTypeEthereum,
		ChainID:                activeChain.ID,
		Metadata:               metadata,
	})

	if err != nil {
		return "", fmt.Errorf("failed to create transaction session: %w", err)
	}

	return sessionID, nil

}
