package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/kaansari/ceerat-platform/ai/ceerat-agent-service/internal/platform"
)

type Agent struct {
	LLM       *OpenAIClient
	Tools     *ToolRunner
	mu        sync.Mutex
	histories map[string][]message
}

func New(llm *OpenAIClient, tools *ToolRunner) *Agent {
	return &Agent{LLM: llm, Tools: tools, histories: map[string][]message{}}
}

func (a *Agent) Chat(ctx context.Context, session *platform.Session, sessionID string, userMessage string) (string, []string, error) {
	if sessionID == "" {
		sessionID = session.UserID
	}

	a.mu.Lock()
	history := sanitizedHistory(a.histories[sessionID])
	a.histories[sessionID] = history
	messages := append([]message{{Role: "system", Content: systemPrompt}}, history...)
	messages = append(messages, message{Role: "user", Content: userMessage})
	a.mu.Unlock()

	actions := []string{}

	for i := 0; i < 4; i++ {
		resp, err := a.LLM.Complete(ctx, messages, toolDefinitions())
		if err != nil {
			return "", actions, err
		}
		messages = append(messages, resp)

		if len(resp.ToolCalls) == 0 {
			a.saveHistory(sessionID, messages)
			return resp.Content, actions, nil
		}

		for _, call := range resp.ToolCalls {
			result, err := a.Tools.Run(ctx, session, call)
			if err != nil {
				result = fmt.Sprintf(`{"error":%q}`, err.Error())
			} else {
				actions = append(actions, call.Function.Name)
			}

			messages = append(messages, message{
				Role:       "tool",
				ToolCallID: call.ID,
				Name:       call.Function.Name,
				Content:    result,
			})
		}
	}

	return "I could not complete the request because it required too many tool steps. Please provide the customer or service details more directly.", actions, nil
}

const systemPrompt = `You are Ceerat Service Agent.

You help authenticated users operate the Ceerat platform.

Capabilities:
- create customers
- list/search customers
- list/search services
- assign services to customers
- create, list, inspect, and update orders
- add services to orders and remove service lines from orders

Rules:
- Never ask the user for their password.
- The user is already authenticated through the platform JWT session.
- Use only tool calls for platform data reads and writes.
- Do not invent customer IDs or service IDs.
- If required fields are missing, ask a concise follow-up question.
- For customer creation, first_name and last_name are required. Other customer fields are optional but should be collected when naturally available.
- To assign a service, identify an existing customer and an existing service. List customers/services if the user provides names instead of IDs.
- To create an order, identify an existing customer and one or more existing services. Use list_customers and list_services before create_order when the user provides names instead of IDs.
- If a customer or service name is ambiguous, ask the user to clarify instead of guessing.
- Never invent order IDs or order service IDs.
- Use status "ordered" unless the user gives another status.
- Summarize completed actions clearly.`

func (a *Agent) saveHistory(sessionID string, messages []message) {
	a.mu.Lock()
	defer a.mu.Unlock()

	withoutSystem := make([]message, 0, len(messages))
	for _, msg := range messages {
		if shouldSaveHistoryMessage(msg) {
			msg.ToolCalls = nil
			msg.ToolCallID = ""
			msg.Name = ""
			withoutSystem = append(withoutSystem, msg)
		}
	}
	if len(withoutSystem) > 20 {
		withoutSystem = withoutSystem[len(withoutSystem)-20:]
	}
	a.histories[sessionID] = withoutSystem
}

func sanitizedHistory(messages []message) []message {
	out := make([]message, 0, len(messages))
	for _, msg := range messages {
		if shouldSaveHistoryMessage(msg) {
			msg.ToolCalls = nil
			msg.ToolCallID = ""
			msg.Name = ""
			out = append(out, msg)
		}
	}
	if len(out) > 20 {
		out = out[len(out)-20:]
	}
	return out
}

func shouldSaveHistoryMessage(msg message) bool {
	if msg.Role == "system" || msg.Role == "tool" || msg.ToolCallID != "" || len(msg.ToolCalls) > 0 {
		return false
	}
	return msg.Role == "user" || msg.Role == "assistant"
}
