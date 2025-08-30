package models

import "time"

// UniswapDeployment represents deployed Uniswap infrastructure contracts
type UniswapDeployment struct {
	ID              uint              `gorm:"primaryKey" json:"id"`
	UserID          *string           `gorm:"index;type:varchar(255)" json:"user_id,omitempty"`
	Version         string            `gorm:"not null" json:"version"`       // v2, v3, v4
	FactoryAddress  string            `json:"factory_address"`               // Uniswap factory contract address
	RouterAddress   string            `json:"router_address"`                // Uniswap router contract address
	WETHAddress     string            `json:"weth_address"`                  // WETH contract address
	DeployerAddress string            `json:"deployer_address"`              // Address that deployed the contracts
	Status          TransactionStatus `gorm:"default:pending" json:"status"` // pending, models.TransactionStatusConfirmed, failed
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`

	ChainID uint  `gorm:"not null" json:"chain_id"`
	Chain   Chain `gorm:"foreignKey:ChainID;references:ID" json:"chain,omitempty"`
}
