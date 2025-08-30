package models

import "time"

type TransactionStatus string

type TransactionChainType string

type TransactionType string

const (
	TransactionChainTypeEthereum TransactionChainType = "ethereum"
	TransactionChainTypeSolana   TransactionChainType = "solana"
)

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusConfirmed TransactionStatus = "confirmed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

const (
	TransactionTypeUniswapV2RouterDeployment  TransactionType = "uniswap_v2_router_deployment"
	TransactionTypeUniswapV2FactoryDeployment TransactionType = "uniswap_v2_factory_deployment"
	TransactionTypeUniswapV2TokenDeployment   TransactionType = "uniswap_v2_token_deployment"
	TransactionTypeLiquidityPoolCreation      TransactionType = "liquidity_pool_creation"
	TransactionTypeLiquidityPoolDeployment    TransactionType = "liquidity_pool_deployment"
	TransactionTypeTokenDeployment            TransactionType = "token_deployment"
	TransactionTypeTokenSwap                  TransactionType = "token_swap"
	TransactionTypeAddLiquidity               TransactionType = "add_liquidity"
	TransactionTypeRemoveLiquidity            TransactionType = "remove_liquidity"
	TransactionTypeRegular                    TransactionType = "regular"
)

type TransactionMetadata struct {
	Key   string `gorm:"not null" json:"key"`
	Value string `gorm:"not null" json:"value"`
}

type TransactionDeployment struct {
	// Title is the title of the transaction used to display in the UI
	Title string `gorm:"not null" json:"title"`
	// Description is the description of the transaction used to display in the UI
	Description string `gorm:"not null" json:"description"`
	// Data is the transaction data included in the transaction body for wallet to sign
	Data string `gorm:"type:text" json:"data"`
	// Value is the value of the transaction for wallet to sign (e.g. 100 WEI)
	Value string `gorm:"not null" json:"value"`
	// Receiver is the receiver of the transaction for wallet to sign (e.g. 0x1234567890123456789012345678901234567890)
	Receiver        string            `gorm:"not null" json:"receiver"`
	Status          TransactionStatus `gorm:"default:pending" json:"status"`
	TransactionType TransactionType   `gorm:"not null" json:"transaction_type"`
}

// TransactionSession represents signing session management
type TransactionSession struct {
	ID                   string                `gorm:"primaryKey" json:"id"`
	UserID               *string               `gorm:"index;type:varchar(255)" json:"user_id,omitempty"`
	Metadata             []TransactionMetadata `gorm:"serializer:json" json:"metadata"`
	TransactionStatus    TransactionStatus     `gorm:"default:pending" json:"status"`
	TransactionChainType TransactionChainType  `gorm:"not null" json:"chain_type"`

	// TransactionDeployments are list of the transactions that needs to be signed
	TransactionDeployments []TransactionDeployment `gorm:"serializer:json" json:"transaction_deployments"`

	ChainID uint  `gorm:"not null" json:"chain_id"`
	Chain   Chain `gorm:"foreignKey:ChainID;references:ID" json:"chain,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
