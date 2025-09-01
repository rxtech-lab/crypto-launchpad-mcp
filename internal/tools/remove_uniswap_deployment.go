package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

type removeUniswapDeploymentTool struct {
	uniswapService services.UniswapService
}

type RemoveUniswapDeploymentArguments struct {
	Ids []uint `json:"ids" validate:"required,min=1"`
}

func NewRemoveUniswapDeploymentTool(uniswapService services.UniswapService) *removeUniswapDeploymentTool {
	return &removeUniswapDeploymentTool{
		uniswapService: uniswapService,
	}
}

func (r *removeUniswapDeploymentTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("remove_uniswap_deployment",
		mcp.WithDescription("Remove one or multiple Uniswap deployments by their IDs. Accepts an array of deployment IDs for bulk removal. This will also clear any related Uniswap configuration settings."),
		mcp.WithArray("ids",
			mcp.Required(),
			mcp.Description("Array of Uniswap deployment IDs to remove. Each ID must be a positive integer."),
			mcp.Items(map[string]any{
				"type":        "number",
				"description": "Uniswap deployment ID",
			}),
		),
	)

	return tool
}

func (r *removeUniswapDeploymentTool) GetHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args RemoveUniswapDeploymentArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		if len(args.Ids) == 0 {
			return mcp.NewToolResultError("No deployment IDs provided"), nil
		}

		// Check if deployments exist before removal
		var existingDeployments []uint
		var deploymentDetails []map[string]any

		for _, id := range args.Ids {
			deployment, err := r.uniswapService.GetUniswapDeployment(id)
			if err == nil && deployment != nil {
				existingDeployments = append(existingDeployments, id)
				deploymentDetails = append(deploymentDetails, map[string]any{
					"id":      deployment.ID,
					"version": deployment.Version,
					"chain":   deployment.ChainID,
					"status":  deployment.Status,
				})
			}
		}

		if len(existingDeployments) == 0 {
			result := map[string]any{
				"requested_ids": args.Ids,
				"removed_count": 0,
				"not_found_ids": args.Ids,
				"message":       "No Uniswap deployments found with the provided IDs",
			}
			resultJSON, _ := json.Marshal(result)
			return mcp.NewToolResultText(fmt.Sprintf("No deployments removed: %s", string(resultJSON))), nil
		}

		// Perform bulk removal using the service method
		if err := r.uniswapService.DeleteUniswapDeployments(existingDeployments); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error removing Uniswap deployments: %v", err)), nil
		}

		removedCount := len(existingDeployments)

		// Find which IDs were not found
		var notFoundIds []uint
		for _, requestedId := range args.Ids {
			found := false
			for _, existingId := range existingDeployments {
				if requestedId == existingId {
					found = true
					break
				}
			}
			if !found {
				notFoundIds = append(notFoundIds, requestedId)
			}
		}

		// Prepare result
		result := map[string]any{
			"requested_ids":       args.Ids,
			"removed_count":       removedCount,
			"removed_ids":         existingDeployments,
			"removed_deployments": deploymentDetails,
		}

		if len(notFoundIds) > 0 {
			result["not_found_ids"] = notFoundIds
		}

		var message string
		if len(args.Ids) == 1 {
			if removedCount == 1 {
				message = "Uniswap deployment removed successfully"
			} else {
				message = "Uniswap deployment not found"
			}
		} else {
			message = fmt.Sprintf("%d Uniswap deployment(s) removed successfully", removedCount)
			if len(notFoundIds) > 0 {
				message += fmt.Sprintf(", %d deployment(s) not found", len(notFoundIds))
			}
		}

		result["message"] = message

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(message + ": "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}
}
