package models

import (
	"time"

	"gorm.io/gorm"
)

type Chain struct {
	ID        uint                 `gorm:"primaryKey" json:"id"`
	ChainType TransactionChainType `gorm:"not null" json:"chain_type"` // ethereum, solana
	RPC       string               `gorm:"not null" json:"rpc"`
	NetworkID string               `gorm:"column:chain_id" json:"chain_id"` // The blockchain's chain ID (e.g., "1" for Ethereum mainnet)
	Name      string               `gorm:"not null" json:"name"`
	IsActive  bool                 `gorm:"default:false" json:"is_active"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
	DeletedAt gorm.DeletedAt       `gorm:"index" json:"-"`
}
