package session

import (
	"sync"
	"time"
)

// Message represents a single message in a session's conversation history.
type Message struct {
	Role      string // "user", "agent", "system"
	Content   string
	Timestamp time.Time
}

// ToolCallRecord tracks a tool invocation made during a session.
type ToolCallRecord struct {
	ID        string
	Title     string
	Kind      string
	Status    string
	Content   string // summary of the result
	Timestamp time.Time
}

// SessionRecord holds the full state of a single agent session including
// its conversation history and tool call records.
type SessionRecord struct {
	ID           string
	AgentName    string
	ConnectionID string
	CWD          string
	Messages     []Message
	ToolCalls    []ToolCallRecord
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Store is an in-memory session store. It manages session records with
// thread-safe access. A future iteration will back this with SQLite.
type Store struct {
	sessions map[string]*SessionRecord
	mu       sync.RWMutex
}

// NewStore creates a new in-memory Store.
func NewStore() *Store {
	return &Store{
		sessions: make(map[string]*SessionRecord),
	}
}

// Create initialises a new SessionRecord and stores it. If a session with the
// given ID already exists it is silently overwritten.
func (s *Store) Create(id, agentName, connectionID, cwd string) *SessionRecord {
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
func (s *Store) Get(id string) *SessionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[id]
}

// AddMessage appends a message to the session's conversation history.
// It is a no-op if the session does not exist.
func (s *Store) AddMessage(sessionID string, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.sessions[sessionID]
	if !ok {
		return
	}

	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	rec.Messages = append(rec.Messages, msg)
	rec.UpdatedAt = time.Now()
}

// AddToolCall appends a tool call record to the session.
// It is a no-op if the session does not exist.
func (s *Store) AddToolCall(sessionID string, tc ToolCallRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.sessions[sessionID]
	if !ok {
		return
	}

	if tc.Timestamp.IsZero() {
		tc.Timestamp = time.Now()
	}

	rec.ToolCalls = append(rec.ToolCalls, tc)
	rec.UpdatedAt = time.Now()
}

// UpdateToolCall finds an existing tool call by ID within the session and
// updates its status and content fields. It is a no-op if the session or
// tool call is not found.
func (s *Store) UpdateToolCall(sessionID, toolCallID, status, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.sessions[sessionID]
	if !ok {
		return
	}

	for i := range rec.ToolCalls {
		if rec.ToolCalls[i].ID == toolCallID {
			rec.ToolCalls[i].Status = status
			rec.ToolCalls[i].Content = content
			rec.UpdatedAt = time.Now()
			return
		}
	}
}

// List returns all session records ordered by creation time (oldest first).
// The returned slice is a snapshot; callers may read but should not modify
// the records without going through Store methods.
func (s *Store) List() []*SessionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*SessionRecord, 0, len(s.sessions))
	for _, rec := range s.sessions {
		out = append(out, rec)
	}
	return out
}

// Delete removes a session from the store. It is a no-op if the session
// does not exist.
func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}
