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
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

// Chain represents blockchain configurations
type Chain struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ChainType string         `gorm:"not null" json:"chain_type"` // ethereum, solana
	RPC       string         `gorm:"not null" json:"rpc"`
	ChainID   string         `json:"chain_id"`
	Name      string         `gorm:"not null" json:"name"`
	IsActive  bool           `gorm:"default:false" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Template represents smart contract templates by chain type
type Template struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"not null" json:"name"`
	Description  string         `json:"description"`
	ChainType    string         `gorm:"not null" json:"chain_type"` // ethereum, solana
	TemplateCode string         `gorm:"type:text;not null" json:"template_code"`
	Metadata     JSON           `gorm:"type:text" json:"metadata"` // Template parameter definitions (key: empty value pairs)
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Deployment represents deployed token contracts
type Deployment struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TemplateID      uint      `gorm:"not null" json:"template_id"`
	ChainID         uint      `gorm:"not null" json:"chain_id"`
	ContractAddress string    `json:"contract_address"`
	TokenName       string    `json:"token_name"`                       // Deprecated: use TemplateValues instead
	TokenSymbol     string    `json:"token_symbol"`                     // Deprecated: use TemplateValues instead
	TemplateValues  JSON      `gorm:"type:text" json:"template_values"` // Runtime template parameter values
	DeployerAddress string    `json:"deployer_address"`
	TransactionHash string    `json:"transaction_hash"`
	Status          string    `gorm:"default:pending" json:"status"` // pending, confirmed, failed
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Template Template `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	Chain    Chain    `gorm:"foreignKey:ChainID;references:ID" json:"chain,omitempty"`
}

// UniswapSettings represents Uniswap version and configuration
type UniswapSettings struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Version         string    `gorm:"not null" json:"version"` // v2, v3, v4
	RouterAddress   string    `json:"router_address"`          // Uniswap router contract address
	FactoryAddress  string    `json:"factory_address"`         // Uniswap factory contract address
	WETHAddress     string    `json:"weth_address"`            // WETH contract address
	QuoterAddress   string    `json:"quoter_address"`          // v3/v4 quoter contract address (optional)
	PositionManager string    `json:"position_manager"`        // v3/v4 position manager address (optional)
	SwapRouter02    string    `json:"swap_router02"`           // v3/v4 SwapRouter02 address (optional)
	IsActive        bool      `gorm:"default:false" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// LiquidityPool represents created pool information
type LiquidityPool struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TokenAddress    string    `gorm:"not null" json:"token_address"`
	PairAddress     string    `gorm:"not null" json:"pair_address"`
	UniswapVersion  string    `gorm:"not null" json:"uniswap_version"`
	Token0          string    `gorm:"not null" json:"token0"`
	Token1          string    `gorm:"not null" json:"token1"`
	InitialToken0   string    `gorm:"not null" json:"initial_token0"`
	InitialToken1   string    `gorm:"not null" json:"initial_token1"`
	CreatorAddress  string    `gorm:"not null" json:"creator_address"`
	TransactionHash string    `gorm:"not null" json:"transaction_hash"`
	Status          string    `gorm:"default:pending" json:"status"` // pending, confirmed, failed
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// LiquidityPosition represents user liquidity positions
type LiquidityPosition struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	PoolID          uint      `gorm:"not null" json:"pool_id"`
	UserAddress     string    `gorm:"not null" json:"user_address"`
	LiquidityAmount string    `gorm:"not null" json:"liquidity_amount"`
	Token0Amount    string    `gorm:"not null" json:"token0_amount"`
	Token1Amount    string    `gorm:"not null" json:"token1_amount"`
	TransactionHash string    `gorm:"not null" json:"transaction_hash"`
	Action          string    `gorm:"not null" json:"action"`        // add, remove
	Status          string    `gorm:"default:pending" json:"status"` // pending, confirmed, failed
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Pool LiquidityPool `gorm:"foreignKey:PoolID" json:"pool,omitempty"`
}

// SwapTransaction represents historical swap data
type SwapTransaction struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserAddress       string    `gorm:"not null" json:"user_address"`
	FromToken         string    `gorm:"not null" json:"from_token"`
	ToToken           string    `gorm:"not null" json:"to_token"`
	FromAmount        string    `gorm:"not null" json:"from_amount"`
	ToAmount          string    `gorm:"not null" json:"to_amount"`
	SlippageTolerance string    `gorm:"not null" json:"slippage_tolerance"`
	TransactionHash   string    `gorm:"not null" json:"transaction_hash"`
	Status            string    `gorm:"default:pending" json:"status"` // pending, confirmed, failed
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// UniswapDeployment represents deployed Uniswap infrastructure contracts
type UniswapDeployment struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ChainID         uint      `gorm:"not null" json:"chain_id"`
	Version         string    `gorm:"not null" json:"version"`       // v2, v3, v4
	FactoryAddress  string    `json:"factory_address"`               // Uniswap factory contract address
	RouterAddress   string    `json:"router_address"`                // Uniswap router contract address
	WETHAddress     string    `json:"weth_address"`                  // WETH contract address
	DeployerAddress string    `json:"deployer_address"`              // Address that deployed the contracts
	FactoryTxHash   string    `json:"factory_tx_hash"`               // Factory deployment transaction hash
	RouterTxHash    string    `json:"router_tx_hash"`                // Router deployment transaction hash
	WETHTxHash      string    `json:"weth_tx_hash"`                  // WETH deployment transaction hash
	Status          string    `gorm:"default:pending" json:"status"` // pending, confirmed, failed
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Chain Chain `gorm:"foreignKey:ChainID;references:ID" json:"chain,omitempty"`
}

// TransactionSession represents signing session management
type TransactionSession struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	SessionType     string    `gorm:"not null" json:"session_type"` // deploy, create_pool, add_liquidity, remove_liquidity, swap, deploy_uniswap, balance_query
	ChainType       string    `gorm:"not null" json:"chain_type"`
	ChainID         string    `gorm:"not null" json:"chain_id"`
	TransactionData string    `gorm:"type:text;not null" json:"transaction_data"` // JSON data for the transaction
	Status          string    `gorm:"default:pending" json:"status"`              // pending, signed, confirmed, failed
	TransactionHash string    `json:"transaction_hash"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}
