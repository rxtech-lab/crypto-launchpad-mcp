package api

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rxtech-lab/launchpad-mcp/internal/assets"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type RPCNetwork struct {
	ChainID string `json:"chain_id"`
	Name    string `json:"name"`
	Rpc     string `json:"rpc"`
}

type TransactionCompleteRequest struct {
	TransactionHash string                   `json:"transactionHash"`
	Status          models.TransactionStatus `json:"status"`
	ContractAddress *string                  `json:"contractAddress,omitempty"`
	SignedMessage   string                   `json:"signedMessage"`
	// Signature is signed by user to prove the ownership
	Signature string `json:"signature"`
}

type ErrorPageData struct {
	Title      string
	Message    string
	StatusCode int
}

// renderErrorPage renders the error HTML template with the provided data
func (s *APIServer) renderErrorPage(c *fiber.Ctx, statusCode int, title, message string) error {
	data := ErrorPageData{
		Title:      title,
		Message:    message,
		StatusCode: statusCode,
	}

	tmpl, err := template.New("error").Parse(string(assets.ErrorHTML))
	if err != nil {
		log.Printf("Error parsing error template: %v", err)
		return c.Status(statusCode).SendString(title)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("Error rendering error template: %v", err)
		return c.Status(statusCode).SendString(title)
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Status(statusCode).Send(buf.Bytes())
}

// handleTransactionPage serves the universal transaction signing page
func (s *APIServer) handleTransactionPage(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	// Get the session from database
	session, err := s.txService.GetTransactionSession(sessionID)
	if err != nil {
		log.Printf("Error getting session %s: %v", sessionID, err)
		return s.renderErrorPage(c, fiber.StatusNotFound, "Session Not Found",
			"The requested transaction session could not be found. This may be because the session has expired, been completed, or the URL is incorrect.")
	}

	if session.TransactionStatus == models.TransactionStatusConfirmed {
		return s.renderErrorPage(c, fiber.StatusNotAcceptable, "Transaction Already Confirmed",
			"This transaction has already been confirmed and completed. No further action is required.")
	}

	// Prepare template data
	data := map[string]interface{}{
		"SessionID": sessionID,
		"RPCNetwork": RPCNetwork{
			ChainID: session.Chain.NetworkID,
			Name:    session.Chain.Name,
			Rpc:     session.Chain.RPC,
		},
		"SigningMessage": utils.GenerateMessage(),
		"SessionData":    session,
	}
	// Render the template with custom functions
	tmplBytes := assets.SigningHTML
	tmpl, err := template.New("signing").Funcs(GetTemplateFuncs()).Parse(string(tmplBytes))
	if err != nil {
		log.Printf("Error parsing signing template: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error parsing template")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("Error rendering signing template: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error rendering template")
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(buf.Bytes())
}

// handleTransactionAPI provides transaction data via API
func (s *APIServer) handleTransactionAPI(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")
	index := c.Params("index")
	body := TransactionCompleteRequest{}
	if err := c.BodyParser(&body); err != nil {
		log.Printf("Error parsing body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	parsedIndex, err := strconv.Atoi(index)
	if err != nil {
		log.Printf("Error parsing index: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid index",
		})
	}

	// Get the session from database
	session, err := s.txService.GetTransactionSession(sessionID)
	if err != nil {
		log.Printf("Error getting session %s: %v", sessionID, err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	// verify the transaction hash
	if err := s.verifyTransactionOnChain(body.TransactionHash, session.Chain); err != nil {
		log.Printf("Error verifying transaction %s: %v", body.TransactionHash, err)
		// update the session status to failed
		session.TransactionStatus = models.TransactionStatusFailed
		if err := s.txService.UpdateTransactionSession(sessionID, session); err != nil {
			log.Printf("Error updating session %s: %v", sessionID, err)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to verify transaction",
		})
	}

	allConfirmed := true
	session.TransactionDeployments[parsedIndex].Status = models.TransactionStatusConfirmed
	deployment := &session.TransactionDeployments[parsedIndex]

	// if all deployments are confirmed, update the session status
	for _, deployment := range session.TransactionDeployments {
		if deployment.Status != models.TransactionStatusConfirmed {
			allConfirmed = false
			break
		}
	}

	if allConfirmed {
		session.TransactionStatus = models.TransactionStatusConfirmed
	}

	// update the session in database
	if err := s.txService.UpdateTransactionSession(sessionID, session); err != nil {
		log.Printf("Error updating session %s: %v", sessionID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update session",
		})
	}
	// use hook
	if err := s.hookService.OnTransactionConfirmed(deployment.TransactionType, body.TransactionHash, *body.ContractAddress, *session); err != nil {
		log.Printf("Error on transaction confirmed: %v", err)
	}
	// Return the session data as JSON
	return c.JSON(body)
}

// verifyTransactionOnChain verifies that a transaction exists and was successful on-chain
func (s *APIServer) verifyTransactionOnChain(txHash string, chain models.Chain) error { // Get active chain configuration to get RPC URL
	// Create RPC client
	rpcClient := utils.NewRPCClient(chain.RPC)
	rpcClient.SetTimeout(15 * time.Second)

	// Verify transaction success
	success, receipt, err := rpcClient.VerifyTransactionSuccess(txHash)
	if err != nil {
		return fmt.Errorf("failed to verify transaction: %w", err)
	}

	if !success {
		return fmt.Errorf("transaction failed on-chain (status: %s)", receipt.Status)
	}

	log.Printf("Transaction %s verified successfully on chain %s (block: %s)", txHash, chain.Name, receipt.BlockNumber)
	return nil
}
