package session

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStore is an in-memory session store used as fallback and in tests.
type MemoryStore struct {
	sessions map[string]*SessionRecord
	mu       sync.RWMutex
}

// NewMemoryStore creates a new in-memory Store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*SessionRecord),
	}
}

// Create initialises a new SessionRecord and stores it. If a session with the
// given ID already exists it is silently overwritten.
func (s *MemoryStore) Create(id, agentName, connectionID, cwd string) *SessionRecord {
	now := time.Now()
	rec := &SessionRecord{
		ID:           id,
		AgentName:    agentName,
		ConnectionID: connectionID,
		CWD:          cwd,
		Messages:     make([]Message, 0),
		ToolCalls:    make([]ToolCallRecord, 0),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.mu.Lock()
	s.sessions[id] = rec
	s.mu.Unlock()

	return rec
}

// Get returns the SessionRecord for the given ID, or nil if not found.
func (s *MemoryStore) Get(id string) *SessionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.sessions[id]
	if !ok {
		return nil
	}

	return cloneSessionRecord(rec)
}

// AddMessage appends a message to the session's conversation history.
// It is a no-op if the session does not exist.
func (s *MemoryStore) AddMessage(sessionID string, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.sessions[sessionID]
	if !ok {
		return
	}

	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	rec.Messages = append(rec.Messages, msg)
	rec.UpdatedAt = time.Now()
}

// AddToolCall appends a tool call record to the session.
// It is a no-op if the session does not exist.
func (s *MemoryStore) AddToolCall(sessionID string, tc ToolCallRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.sessions[sessionID]
	if !ok {
		return
	}

	if tc.Timestamp.IsZero() {
		tc.Timestamp = time.Now()
	}
	tc.Parts = append([]ToolCallPart(nil), tc.Parts...)

	rec.ToolCalls = append(rec.ToolCalls, tc)
	rec.UpdatedAt = time.Now()
}

// UpdateToolCall finds an existing tool call by ID within the session and
// updates its status and content fields. It is a no-op if the session or
// tool call is not found.
func (s *MemoryStore) UpdateToolCall(
	sessionID,
	toolCallID,
	status,
	content string,
	parts []ToolCallPart,
	diffSummary ToolCallDiffSummary,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.sessions[sessionID]
	if !ok {
		return
	}

	for i := range rec.ToolCalls {
		if rec.ToolCalls[i].ID == toolCallID {
			rec.ToolCalls[i].Status = status
			if content != "" {
				rec.ToolCalls[i].Content = content
			}
			rec.ToolCalls[i].Parts = append([]ToolCallPart(nil), parts...)
			rec.ToolCalls[i].DiffSummary = diffSummary
			rec.UpdatedAt = time.Now()
			return
		}
	}
}

// List returns all session records.
func (s *MemoryStore) List() []*SessionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*SessionRecord, 0, len(s.sessions))
	for _, rec := range s.sessions {
		out = append(out, cloneSessionRecord(rec))
	}
	return out
}

// Delete removes a session from the store.
func (s *MemoryStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// Close satisfies Store.
func (s *MemoryStore) Close() error {
	return nil
}

func cloneSessionRecord(rec *SessionRecord) *SessionRecord {
	if rec == nil {
		return nil
	}

	copyRec := *rec
	copyRec.Messages = append([]Message(nil), rec.Messages...)
	copyRec.ToolCalls = append([]ToolCallRecord(nil), rec.ToolCalls...)
	for i := range copyRec.ToolCalls {
		copyRec.ToolCalls[i].Parts = append([]ToolCallPart(nil), copyRec.ToolCalls[i].Parts...)
	}
	return &copyRec
}
