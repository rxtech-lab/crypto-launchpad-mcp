package models

import "time"

type Deployment struct {
	ID              uint              `gorm:"primaryKey" json:"id"`
	UserID          *string           `gorm:"index;type:varchar(255)" json:"user_id,omitempty"`
	TemplateID      uint              `gorm:"not null" json:"template_id"`
	ChainID         uint              `gorm:"not null" json:"chain_id"`
	ContractAddress string            `json:"contract_address"`
	TemplateValues  JSON              `gorm:"type:text" json:"template_values"` // Runtime template parameter values
	DeployerAddress string            `json:"deployer_address"`
	TransactionHash string            `json:"transaction_hash"`
	Status          TransactionStatus `gorm:"default:pending" json:"status"` // pending, models.TransactionStatusConfirmed, failed
	SessionId       string            `gorm:"index" json:"session_id"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`

	Template Template           `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	Chain    Chain              `gorm:"foreignKey:ChainID;references:ID" json:"chain,omitempty"`
	Session  TransactionSession `gorm:"foreignKey:SessionId;references:ID" json:"session,omitempty"`
}
