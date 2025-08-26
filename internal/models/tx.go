package models

import "time"

type TransactionStatus string

type TransactionChainType string

const (
	TransactionChainTypeEthereum TransactionChainType = "ethereum"
	TransactionChainTypeSolana   TransactionChainType = "solana"
)

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusConfirmed TransactionStatus = "confirmed"
	TransactionStatusFailed    TransactionStatus = "failed"
)


type TransactionMetadata struct {
	Key   string `gorm:"not null" json:"key"`
	Value string `gorm:"not null" json:"value"`
}


type TransactionDeployment struct {
  Title string `gorm:"not null" json:"title"`
  Description string `gorm:"not null" json:"description"`
  Data string `gorm:"type:text" json:"data"` // transaction data included in the transaction body
  Value string `gorm:"not null" json:"value"` // value of the transaction
}

// TransactionSession represents signing session management
type TransactionSession struct {
	ID              string            `gorm:"primaryKey" json:"id"`
	Metadata  []TransactionMetadata `gorm:"type:text" json:"metadata"`
    TransactionStatus TransactionStatus `gorm:"default:pending" json:"status"`
	TransactionChainType TransactionChainType `gorm:"not null" json:"chain_type"`

	// TransactionDeployments are list of the transactions that needs to be signed
	TransactionDeployments []TransactionDeployment `gorm:"type:text" json:"transaction_deployments"`

	Chain Chain `gorm:"foreignKey:ChainID;references:ID" json:"chain,omitempty"`
	ChainID uint `gorm:"not null" json:"chain_id"`

	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	ExpiresAt       time.Time         `json:"expires_at"`
}