package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/rxtech-lab/launchpad-mcp/internal/contracts"
)

// handleContractArtifact serves contract artifacts (ABI and bytecode)
func (s *APIServer) handleContractArtifact(c *fiber.Ctx) error {
	contractName := c.Params("name")
	if contractName == "" {
		return c.Status(400).JSON(map[string]string{
			"error": "Contract name is required",
		})
	}

	artifact, err := contracts.GetContractArtifact(contractName)
	if err != nil {
		log.Printf("Error getting contract artifact for %s: %v", contractName, err)
		return c.Status(404).JSON(map[string]string{
			"error": "Contract not found",
		})
	}

	return c.JSON(artifact)
}
