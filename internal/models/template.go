package models

import (
	"time"

	"gorm.io/gorm"
)

type Template struct {
	ID                   uint                 `gorm:"primaryKey" json:"id"`
	Name                 string               `gorm:"not null" json:"name"`
	Description          string               `json:"description"`
	UserId               *string              `gorm:"index;type:varchar(255)" json:"user_id,omitempty"`
	ChainType            TransactionChainType `gorm:"not null" json:"chain_type"` // ethereum, solana
	TemplateCode         string               `gorm:"type:text;not null" json:"template_code"`
	Metadata             JSON                 `gorm:"type:text" json:"metadata"` // Template parameter definitions (key: empty value pairs)
	SampleTemplateValues JSON                 `gorm:"type:text" json:"sample_template_values"`
	Abi                  JSON                 `gorm:"type:text" json:"abi"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
	DeletedAt            gorm.DeletedAt       `gorm:"index" json:"-"`
}
