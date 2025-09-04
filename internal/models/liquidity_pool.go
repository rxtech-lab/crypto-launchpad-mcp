package models

import "time"

type LiquidityPool struct {
	ID              uint              `gorm:"primaryKey" json:"id"`
	UserID          *string           `gorm:"index;type:varchar(255)" json:"user_id,omitempty"`
	TokenAddress    string            `gorm:"not null" json:"token_address"`
	PairAddress     string            `gorm:"not null" json:"pair_address"`
	UniswapVersion  string            `gorm:"not null" json:"uniswap_version"`
	Token0          string            `gorm:"not null" json:"token0"`
	Token1          string            `gorm:"not null" json:"token1"`
	InitialToken0   string            `gorm:"not null" json:"initial_token0"`
	InitialToken1   string            `gorm:"not null" json:"initial_token1"`
	CreatorAddress  string            `gorm:"not null" json:"creator_address"`
	TransactionHash string            `gorm:"not null" json:"transaction_hash"`
	Status          TransactionStatus `gorm:"default:pending" json:"status"` // pending, models.TransactionStatusConfirmed, failed
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`

	SessionId string             `gorm:"index" json:"session_id"`
	Session   TransactionSession `gorm:"foreignKey:SessionId;references:ID" json:"session,omitempty"`
}
