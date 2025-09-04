package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type JSON map[string]interface{}

// Value Implement the driver.Valuer interface for JSON type
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan Implement the sql.Scanner interface for JSON type
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
