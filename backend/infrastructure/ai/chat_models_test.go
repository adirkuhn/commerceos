package ai_test

import (
	"github.com/adirkuhn/commerceos/domain/assistant"
)

// sampleTurn is one conversation every assistant.ChatModel adapter must
// translate: a question, a tool call and its result, an app-written notice,
// and a follow-up.
func sampleTurn() assistant.ChatTurn {
	return assistant.ChatTurn{
		System: "be helpful",
		Messages: []assistant.Message{
			{Role: assistant.RoleUser, Content: "status CO-12?"},
			{Role: assistant.RoleAssistant, Content: "Looking it up.", ToolCalls: []assistant.ToolCall{{ID: "call_1", Name: "get_sales_order", Arguments: `{"order":"CO-12"}`}}},
			{Role: assistant.RoleTool, ToolCallID: "call_1", ToolName: "get_sales_order", Content: `{"status":"shipped"}`},
			{Role: assistant.RoleAssistant, Question: true, Notice: assistant.NoticeTooManySteps},
			{Role: assistant.RoleUser, Content: "and CO-13?"},
		},
		Tools: []assistant.ToolDefinition{orderTool},
	}
}
