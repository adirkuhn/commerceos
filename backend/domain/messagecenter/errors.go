package messagecenter

import "errors"

var (
	ErrTemplateNotFound     = errors.New("message template not found")
	ErrTemplateNameRequired = errors.New("template name is required")
	ErrInvalidCategory      = errors.New("invalid template category")
	ErrNoChannelsSelected   = errors.New("at least one channel must be selected")
	ErrInvalidChannel       = errors.New("invalid message channel")
	ErrTemplateBodyRequired = errors.New("template body is required")

	ErrRuleNotFound           = errors.New("automation rule not found")
	ErrRuleNameRequired       = errors.New("rule name is required")
	ErrInvalidTriggerSubject  = errors.New("invalid trigger subject")
	ErrTemplateIDRequired     = errors.New("template id is required")
	ErrInvalidCondition       = errors.New("invalid condition")
	ErrInvalidExtraRecipient  = errors.New("extra recipient is not a valid email address")
	ErrTooManyExtraRecipients = errors.New("at most 20 extra recipients per rule")
)
