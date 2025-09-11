package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

type viewTemplateTool struct {
	templateService services.TemplateService
	evmService      services.EvmService
}

type ViewTemplateArguments struct {
	TemplateID string `json:"template_id" validate:"required"`
}

type ViewTemplateResult struct {
	ID          uint                        `json:"id"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	ChainType   models.TransactionChainType `json:"chain_type"`
	AbiMethods  []AbiMethodInfo             `json:"abi_methods,omitempty"`
	AbiMethod   *AbiMethodDetail            `json:"abi_method,omitempty"`
}

type AbiMethodInfo struct {
	Name    string              `json:"name"`
	Type    string              `json:"type"`
	Inputs  []AbiMethodArgument `json:"inputs"`
	Outputs []AbiMethodArgument `json:"outputs"`
}

type AbiMethodDetail struct {
	Name    string              `json:"name"`
	Type    string              `json:"type"`
	Inputs  []AbiMethodArgument `json:"inputs"`
	Outputs []AbiMethodArgument `json:"outputs"`
}

type AbiMethodArgument struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func NewViewTemplateTool(templateService services.TemplateService, evmService services.EvmService) *viewTemplateTool {
	return &viewTemplateTool{
		templateService: templateService,
		evmService:      evmService,
	}
}

func (c *viewTemplateTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("view_template",
		mcp.WithDescription("View the template by id. The list_template tool only returns a summary of templates. Use this tool to get detailed information including all available methods and method parameters."),
		mcp.WithString("template_id",
			mcp.Required(),
			mcp.Description("ID of the template to view"),
		),
		mcp.WithBoolean("show_abi_methods",
			mcp.Description("If true, will show the abi methods available in the template."),
		),
		mcp.WithString("abi_method",
			mcp.Description("Name of the abi method that you are interested in. Leave it blank if show_abi_methods is false or you want to see all the abi methods available in the template."),
		),
	)

	return tool
}

func (c *viewTemplateTool) GetHandler() server.ToolHandlerFunc {

	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse and validate arguments
		var args ViewTemplateArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		// Parse template ID to uint
		templateID := uint(0)
		if _, err := fmt.Sscanf(args.TemplateID, "%d", &templateID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid template_id format: %v", err)), nil
		}

		// Fetch template from database
		template, err := c.templateService.GetTemplateByID(templateID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Template not found: %v", err)), nil
		}

		// Prepare basic result
		result := ViewTemplateResult{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			ChainType:   template.ChainType,
		}

		// Handle show_abi_methods parameter
		showAbiMethods := false
		if arguments, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if val, exists := arguments["show_abi_methods"]; exists {
				if boolVal, ok := val.(bool); ok {
					showAbiMethods = boolVal
				}
			}
		}

		// Handle abi_method parameter
		abiMethodName := ""
		if arguments, ok := request.Params.Arguments.(map[string]interface{}); ok {
			if val, exists := arguments["abi_method"]; exists {
				if strVal, ok := val.(string); ok && strVal != "" {
					abiMethodName = strVal
				}
			}
		}

		// Process ABI information if template has ABI and user requested it
		if template.Abi != nil && (showAbiMethods || abiMethodName != "") {
			if abiMethodName != "" {
				// Show specific method
				method, err := c.evmService.GetAbiMethod(template.Abi, abiMethodName)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("ABI method not found: %v", err)), nil
				}

				// Convert method to our format
				methodDetail := &AbiMethodDetail{
					Name:    method.Name,
					Type:    functionTypeToString(method.Type),
					Inputs:  convertAbiArguments(method.Inputs),
					Outputs: convertAbiArguments(method.Outputs),
				}
				result.AbiMethod = methodDetail

			} else if showAbiMethods {
				// Show all methods
				methods, err := c.evmService.GetAllAbiMethods(template.Abi)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Error retrieving ABI methods: %v", err)), nil
				}

				// Convert methods to our format
				var methodInfos []AbiMethodInfo
				for _, method := range methods {
					methodInfo := AbiMethodInfo{
						Name:    method.Name,
						Type:    functionTypeToString(method.Type),
						Inputs:  convertAbiArguments(method.Inputs),
						Outputs: convertAbiArguments(method.Outputs),
					}
					methodInfos = append(methodInfos, methodInfo)
				}
				result.AbiMethods = methodInfos
			}
		}

		// Format success message and return result
		successMessage := fmt.Sprintf("Template '%s' retrieved successfully", template.Name)

		// Add ABI information to the message
		if result.AbiMethod != nil {
			successMessage += fmt.Sprintf(" (showing method: %s)", result.AbiMethod.Name)
		} else if len(result.AbiMethods) > 0 {
			successMessage += fmt.Sprintf(" (showing %d ABI methods)", len(result.AbiMethods))
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(successMessage + ": "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}
}

// Helper function to convert ethereum ABI arguments to our format
func convertAbiArguments(abiArgs abi.Arguments) []AbiMethodArgument {
	args := make([]AbiMethodArgument, 0) // Ensure we always return a non-nil slice
	for _, arg := range abiArgs {
		args = append(args, AbiMethodArgument{
			Name: arg.Name,
			Type: arg.Type.String(),
		})
	}
	return args
}

// Helper function to convert FunctionType to string
func functionTypeToString(funcType abi.FunctionType) string {
	switch funcType {
	case abi.Constructor:
		return "constructor"
	case abi.Fallback:
		return "fallback"
	case abi.Receive:
		return "receive"
	case abi.Function:
		return "function"
	default:
		return "unknown"
	}
}
