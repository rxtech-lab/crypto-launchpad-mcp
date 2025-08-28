package services

import "github.com/rxtech-lab/launchpad-mcp/internal/models"

// Hook is used to perform actions when a transaction is confirmed base on their transaction type
type Hook interface {
	// CanHandle is used to check if the hook can handle the transaction type
	CanHandle(txType models.TransactionType) bool
	// OnTransactionConfirmed is called when a transaction is confirmed
	OnTransactionConfirmed(txType models.TransactionType, txHash string, contractAddress string, session models.TransactionSession) error
}
