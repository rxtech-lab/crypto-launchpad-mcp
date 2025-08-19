package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
)

func NewListDeploymentsTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("list_deployments",
		mcp.WithDescription("List all token deployments with pagination support and filtering options including status, contract addresses, and transaction hashes."),
		mcp.WithString("status",
			mcp.Description("Filter by deployment status (pending, models.TransactionStatusConfirmed, failed). Leave empty to get all deployments"),
		),
		mcp.WithString("chain_type",
			mcp.Description("Filter by blockchain type (ethereum, solana). Leave empty to get deployments from all chains"),
		),
		mcp.WithString("page",
			mcp.Description("Page number for pagination (default: 1)"),
		),
		mcp.WithString("limit",
			mcp.Description("Number of deployments per page (default: 10, max: 100)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		status := request.GetString("status", "")
		chainType := request.GetString("chain_type", "")
		pageStr := request.GetString("page", "1")
		limitStr := request.GetString("limit", "10")

		// Parse pagination parameters
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}

		// Get all deployments from database
		deployments, err := db.ListDeployments()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error retrieving deployments: %v", err)),
				},
			}, nil
		}

		// Filter deployments based on parameters
		var filteredDeployments []interface{}
		for _, deployment := range deployments {
			// Apply status filter
			if status != "" && deployment.Status != status {
				continue
			}

			// Apply chain type filter
			if chainType != "" && deployment.Chain.ChainType != chainType {
				continue
			}

			// Convert to map for JSON output
			deploymentData := map[string]interface{}{
				"id":               deployment.ID,
				"template_id":      deployment.TemplateID,
				"contract_address": deployment.ContractAddress,
				"token_name":       deployment.TokenName,
				"token_symbol":     deployment.TokenSymbol,
				"chain_id":         deployment.ChainID,
				"chain":            deployment.Chain,
				"deployer_address": deployment.DeployerAddress,
				"transaction_hash": deployment.TransactionHash,
				"status":           deployment.Status,
				"created_at":       deployment.CreatedAt,
				"updated_at":       deployment.UpdatedAt,
			}

			// Include template information if available
			if deployment.Template.ID != 0 {
				deploymentData["template"] = map[string]interface{}{
					"id":          deployment.Template.ID,
					"name":        deployment.Template.Name,
					"description": deployment.Template.Description,
					"chain_type":  deployment.Template.ChainType,
				}
			}

			filteredDeployments = append(filteredDeployments, deploymentData)
		}

		// Calculate pagination
		totalCount := len(filteredDeployments)
		totalPages := (totalCount + limit - 1) / limit
		startIndex := (page - 1) * limit
		endIndex := startIndex + limit

		// Apply pagination to results
		var paginatedDeployments []interface{}
		if startIndex < totalCount {
			if endIndex > totalCount {
				endIndex = totalCount
			}
			paginatedDeployments = filteredDeployments[startIndex:endIndex]
		}

		result := map[string]interface{}{
			"deployments": paginatedDeployments,
			"pagination": map[string]interface{}{
				"current_page": page,
				"total_pages":  totalPages,
				"page_size":    limit,
				"total_count":  totalCount,
				"has_next":     page < totalPages,
				"has_previous": page > 1,
			},
			"filters": map[string]interface{}{
				"status":     status,
				"chain_type": chainType,
			},
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Deployments list: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
