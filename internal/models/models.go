package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// JSON is a custom type for JSON fields
type JSON map[string]interface{}

// Implement the driver.Valuer interface for JSON type
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Implement the sql.Scanner interface for JSON type
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	case nil:
		*j = nil
		return nil
	default:
		return errors.New("type assertion to []byte failed")
	}

	if len(bytes) == 0 {
		*j = nil
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// Chain represents blockchain configurations
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

// Template represents smart contract templates by chain type

// Deployment represents deployed token contracts
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

// LiquidityPool represents created pool information
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
}

// LiquidityPosition represents user liquidity positions
type LiquidityPosition struct {
	ID              uint              `gorm:"primaryKey" json:"id"`
	UserID          *string           `gorm:"index;type:varchar(255)" json:"user_id,omitempty"`
	PoolID          uint              `gorm:"not null" json:"pool_id"`
	UserAddress     string            `gorm:"not null" json:"user_address"`
	LiquidityAmount string            `gorm:"not null" json:"liquidity_amount"`
	Token0Amount    string            `gorm:"not null" json:"token0_amount"`
	Token1Amount    string            `gorm:"not null" json:"token1_amount"`
	TransactionHash string            `gorm:"not null" json:"transaction_hash"`
	Action          string            `gorm:"not null" json:"action"`        // add, remove
	Status          TransactionStatus `gorm:"default:pending" json:"status"` // pending, models.TransactionStatusConfirmed, failed
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`

	Pool LiquidityPool `gorm:"foreignKey:PoolID" json:"pool,omitempty"`
}

// SwapTransaction represents historical swap data
type SwapTransaction struct {
	ID                uint              `gorm:"primaryKey" json:"id"`
	UserID            *string           `gorm:"index;type:varchar(255)" json:"user_id,omitempty"`
	UserAddress       string            `gorm:"not null" json:"user_address"`
	FromToken         string            `gorm:"not null" json:"from_token"`
	ToToken           string            `gorm:"not null" json:"to_token"`
	FromAmount        string            `gorm:"not null" json:"from_amount"`
	ToAmount          string            `gorm:"not null" json:"to_amount"`
	SlippageTolerance string            `gorm:"not null" json:"slippage_tolerance"`
	TransactionHash   string            `gorm:"not null" json:"transaction_hash"`
	Status            TransactionStatus `gorm:"default:pending" json:"status"` // pending, models.TransactionStatusConfirmed, failed
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}
