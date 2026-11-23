package events

// Admin AI assistant (backend/application/assistant). What the user wrote
// and what the model answered are never in an event -- read the request's
// messages by entity_id.
var (
	AssistantRequestCreated = defineAction("assistant.request.created",
		"An admin asked the AI assistant something; the request is queued for the model.")
	AssistantRequestReplied = defineAction("assistant.request.replied",
		"An admin answered the assistant's question or asked a follow-up; the request is queued again.")
	AssistantRequestAnswered = defineAction("assistant.request.answered",
		"The assistant's model answered a request. data: attempts, tool_calls.")
	AssistantRequestQuestionAsked = defineAction("assistant.request.question_asked",
		"The assistant's model asked the admin to clarify a request. data: attempts, tool_calls, notice.")
	AssistantRequestCancelled = defineAction("assistant.request.cancelled",
		"An admin cancelled an assistant request. data: previous_status.")
)
