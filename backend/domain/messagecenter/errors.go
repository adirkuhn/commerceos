package messagecenter

import "errors"

var (
	ErrTemplateNotFound     = errors.New("message template not found")
	ErrTemplateNameRequired = errors.New("template name is required")
	ErrInvalidCategory      = errors.New("invalid template category")
	ErrNoChannelsSelected   = errors.New("at least one channel must be selected")
	ErrInvalidChannel       = errors.New("invalid message channel")
	ErrTemplateBodyRequired = errors.New("template body is required")
)
