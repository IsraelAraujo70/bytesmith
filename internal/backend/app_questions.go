package backend

import (
	"strings"

	"bytesmith/internal/acp"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// Explicit user-input questions (item/tool/requestUserInput)
// ---------------------------------------------------------------------------

// RespondQuestion is called by the UI when the user submits answers to an
// explicit question request.
func (a *App) RespondQuestion(requestID string, answers map[string][]string) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return
	}

	response := emptyQuestionResponse()
	for questionID, values := range answers {
		questionID = strings.TrimSpace(questionID)
		if questionID == "" {
			continue
		}

		result := make([]string, 0, len(values))
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				continue
			}
			result = append(result, value)
		}
		response.Answers[questionID] = acp.ToolRequestUserInputAnswer{Answers: result}
	}

	a.answerQuestionRequest(requestID, response)
}

// RejectQuestion is called by the UI when the user dismisses a question
// request.
func (a *App) RejectQuestion(requestID string) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return
	}
	a.answerQuestionRequest(requestID, emptyQuestionResponse())
}

func (a *App) handleQuestionRequest(connectionID string, params acp.ToolRequestUserInputParams) acp.ToolRequestUserInputResponse {
	requestID := uuid.NewString()
	ch := make(chan acp.ToolRequestUserInputResponse, 1)

	a.pendingQuestionsMu.Lock()
	a.pendingQuestions[requestID] = ch
	a.pendingQuestionsMu.Unlock()

	questions := make([]QuestionInfo, 0, len(params.Questions))
	for _, q := range params.Questions {
		options := make([]QuestionOptionInfo, 0, len(q.Options))
		for _, opt := range q.Options {
			options = append(options, QuestionOptionInfo{
				Label:       opt.Label,
				Description: opt.Description,
			})
		}

		questions = append(questions, QuestionInfo{
			ID:       q.ID,
			Header:   q.Header,
			Question: q.Question,
			Multiple: q.Multiple,
			IsOther:  q.IsOther,
			IsSecret: q.IsSecret,
			Options:  options,
		})
	}

	wailsRuntime.EventsEmit(a.ctx, "agent:question", QuestionRequestInfo{
		RequestID:    requestID,
		ConnectionID: connectionID,
		SessionID:    params.ThreadID,
		ToolCallID:   params.ItemID,
		Questions:    questions,
	})

	response, ok := <-ch

	a.pendingQuestionsMu.Lock()
	delete(a.pendingQuestions, requestID)
	a.pendingQuestionsMu.Unlock()

	if !ok {
		return emptyQuestionResponse()
	}
	if response.Answers == nil {
		response.Answers = map[string]acp.ToolRequestUserInputAnswer{}
	}
	return response
}

func (a *App) answerQuestionRequest(requestID string, response acp.ToolRequestUserInputResponse) {
	a.pendingQuestionsMu.Lock()
	ch, ok := a.pendingQuestions[requestID]
	a.pendingQuestionsMu.Unlock()
	if !ok {
		return
	}

	select {
	case ch <- response:
	default:
	}
}

func emptyQuestionResponse() acp.ToolRequestUserInputResponse {
	return acp.ToolRequestUserInputResponse{
		Answers: map[string]acp.ToolRequestUserInputAnswer{},
	}
}
