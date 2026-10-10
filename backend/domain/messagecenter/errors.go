package messagecenter

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrTemplateNotFound     = errs.New("MESSAGE_TEMPLATE_NOT_FOUND", "message template not found")
	ErrTemplateNameRequired = errs.New("MESSAGE_TEMPLATE_NAME_REQUIRED", "template name is required")
	ErrInvalidCategory      = errs.New("MESSAGE_TEMPLATE_INVALID_CATEGORY", "invalid template category")
	ErrNoChannelsSelected   = errs.New("MESSAGE_TEMPLATE_NO_CHANNELS", "at least one channel must be selected")
	ErrInvalidChannel       = errs.New("MESSAGE_INVALID_CHANNEL", "invalid message channel")
	ErrTemplateTextRequired = errs.New("MESSAGE_TEMPLATE_TEXT_REQUIRED", "a text message is required for WhatsApp/SMS")
	ErrTemplateBodyRequired = errs.New("MESSAGE_TEMPLATE_BODY_REQUIRED", "template body is required")

	ErrRuleNotFound           = errs.New("AUTOMATION_RULE_NOT_FOUND", "automation rule not found")
	ErrRuleNameRequired       = errs.New("AUTOMATION_RULE_NAME_REQUIRED", "rule name is required")
	ErrInvalidTriggerSubject  = errs.New("AUTOMATION_RULE_INVALID_TRIGGER", "invalid trigger subject")
	ErrTemplateIDRequired     = errs.New("AUTOMATION_RULE_TEMPLATE_REQUIRED", "template id is required")
	ErrInvalidCondition       = errs.New("AUTOMATION_RULE_INVALID_CONDITION", "invalid condition")
	ErrInvalidExtraRecipient  = errs.New("AUTOMATION_RULE_INVALID_EXTRA_RECIPIENT", "extra recipient is not a valid email address")
	ErrTooManyExtraRecipients = errs.New("AUTOMATION_RULE_TOO_MANY_EXTRA_RECIPIENTS", "at most 20 extra recipients per rule")
)
