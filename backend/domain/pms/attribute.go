package pms

import "time"

type AttributeType string

const (
	AttributeTypeText    AttributeType = "text"
	AttributeTypeNumber  AttributeType = "number"
	AttributeTypeBoolean AttributeType = "boolean"
	AttributeTypeSelect  AttributeType = "select"
)

type AttributeDefinition struct {
	ID        string
	TenantID  string
	Name      string
	Type      AttributeType
	Options   []string // non-nil only when Type == AttributeTypeSelect
	CreatedAt time.Time
}
