package session

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// SQLiteStore persists sessions to a local sqlite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or creates the sqlite session database.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("session: empty sqlite path")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("session: create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("session: open sqlite: %w", err)
	}

	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("session: enable wal: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("session: enable foreign keys: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			agent_name TEXT NOT NULL,
			connection_id TEXT NOT NULL,
			cwd TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session_ts ON messages(session_id, timestamp);`,
		`CREATE TABLE IF NOT EXISTS tool_calls (
			row_id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			tool_call_id TEXT NOT NULL,
			title TEXT NOT NULL,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			UNIQUE(session_id, tool_call_id),
			FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_session_ts ON tool_calls(session_id, timestamp);`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("session: migrate: %w", err)
		}
	}

	return nil
}

// Create upserts a session record in sqlite.
func (s *SQLiteStore) Create(id, agentName, connectionID, cwd string) *SessionRecord {
	now := time.Now().UTC()
	nowS := now.Format(time.RFC3339Nano)
	_, _ = s.db.Exec(
		`INSERT INTO sessions (id, agent_name, connection_id, cwd, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   agent_name=excluded.agent_name,
		   connection_id=excluded.connection_id,
		   cwd=excluded.cwd,
		   updated_at=excluded.updated_at`,
		id, agentName, connectionID, cwd, nowS, nowS,
	)

	return &SessionRecord{
		ID:           id,
		AgentName:    agentName,
		ConnectionID: connectionID,
		CWD:          cwd,
		Messages:     []Message{},
		ToolCalls:    []ToolCallRecord{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Get returns the full session record with messages and tool calls.
func (s *SQLiteStore) Get(id string) *SessionRecord {
	row := s.db.QueryRow(
		`SELECT id, agent_name, connection_id, cwd, created_at, updated_at
		 FROM sessions WHERE id = ?`,
		id,
	)

	var rec SessionRecord
	var createdS, updatedS string
	if err := row.Scan(&rec.ID, &rec.AgentName, &rec.ConnectionID, &rec.CWD, &createdS, &updatedS); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return nil
	}

	rec.CreatedAt = parseRFC3339(createdS)
	rec.UpdatedAt = parseRFC3339(updatedS)
	rec.Messages = s.messagesForSession(id)
	rec.ToolCalls = s.toolCallsForSession(id)
	return &rec
}

// AddMessage appends one message and bumps session updated_at.
func (s *SQLiteStore) AddMessage(sessionID string, msg Message) {
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}
	ts := msg.Timestamp.UTC().Format(time.RFC3339Nano)

	tx, err := s.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO messages (id, session_id, role, content, timestamp)
		 VALUES (?, ?, ?, ?, ?)`,
		msg.ID, sessionID, msg.Role, msg.Content, ts,
	); err != nil {
		return
	}

	if _, err := tx.Exec(
		`UPDATE sessions SET updated_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339Nano), sessionID,
	); err != nil {
		return
	}

	_ = tx.Commit()
}

// AddToolCall inserts or replaces a tool call record.
func (s *SQLiteStore) AddToolCall(sessionID string, tc ToolCallRecord) {
	if tc.Timestamp.IsZero() {
		tc.Timestamp = time.Now().UTC()
	}
	ts := tc.Timestamp.UTC().Format(time.RFC3339Nano)
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := s.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO tool_calls (session_id, tool_call_id, title, kind, status, content, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(session_id, tool_call_id) DO UPDATE SET
		   title=excluded.title,
		   kind=excluded.kind,
		   status=excluded.status,
		   content=excluded.content,
		   timestamp=excluded.timestamp`,
		sessionID, tc.ID, tc.Title, tc.Kind, tc.Status, tc.Content, ts,
	); err != nil {
		return
	}

	if _, err := tx.Exec(`UPDATE sessions SET updated_at = ? WHERE id = ?`, now, sessionID); err != nil {
		return
	}

	_ = tx.Commit()
}

// UpdateToolCall updates status/content for one tool call.
func (s *SQLiteStore) UpdateToolCall(sessionID, toolCallID, status, content string) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if content == "" {
		_, _ = s.db.Exec(
			`UPDATE tool_calls
			 SET status = ?, timestamp = ?
			 WHERE session_id = ? AND tool_call_id = ?`,
			status, now, sessionID, toolCallID,
		)
	} else {
		_, _ = s.db.Exec(
			`UPDATE tool_calls
			 SET status = ?, content = ?, timestamp = ?
			 WHERE session_id = ? AND tool_call_id = ?`,
			status, content, now, sessionID, toolCallID,
		)
	}
	_, _ = s.db.Exec(`UPDATE sessions SET updated_at = ? WHERE id = ?`, now, sessionID)
}

// List returns every session with full messages and tool calls.
func (s *SQLiteStore) List() []*SessionRecord {
	rows, err := s.db.Query(
		`SELECT id, agent_name, connection_id, cwd, created_at, updated_at
		 FROM sessions`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]*SessionRecord, 0)
	for rows.Next() {
		var rec SessionRecord
		var createdS, updatedS string
		if err := rows.Scan(&rec.ID, &rec.AgentName, &rec.ConnectionID, &rec.CWD, &createdS, &updatedS); err != nil {
			continue
		}
		rec.CreatedAt = parseRFC3339(createdS)
		rec.UpdatedAt = parseRFC3339(updatedS)
		rec.Messages = s.messagesForSession(rec.ID)
		rec.ToolCalls = s.toolCallsForSession(rec.ID)
		result = append(result, &rec)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

// Delete removes a session and all child rows.
func (s *SQLiteStore) Delete(id string) {
	_, _ = s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
}

// Close closes the underlying sqlite handle.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) messagesForSession(sessionID string) []Message {
	rows, err := s.db.Query(
		`SELECT id, role, content, timestamp
		 FROM messages
		 WHERE session_id = ?
		 ORDER BY timestamp ASC`,
		sessionID,
	)
	if err != nil {
		return []Message{}
	}
	defer rows.Close()

	out := make([]Message, 0)
	for rows.Next() {
		var m Message
		var ts string
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &ts); err != nil {
			continue
		}
		m.Timestamp = parseRFC3339(ts)
		out = append(out, m)
	}
	return out
}

func (s *SQLiteStore) toolCallsForSession(sessionID string) []ToolCallRecord {
	rows, err := s.db.Query(
		`SELECT tool_call_id, title, kind, status, content, timestamp
		 FROM tool_calls
		 WHERE session_id = ?
		 ORDER BY timestamp ASC`,
		sessionID,
	)
	if err != nil {
		return []ToolCallRecord{}
	}
	defer rows.Close()

	out := make([]ToolCallRecord, 0)
	for rows.Next() {
		var tc ToolCallRecord
		var ts string
		if err := rows.Scan(&tc.ID, &tc.Title, &tc.Kind, &tc.Status, &tc.Content, &ts); err != nil {
			continue
		}
		tc.Timestamp = parseRFC3339(ts)
		out = append(out, tc)
	}
	return out
}

func parseRFC3339(v string) time.Time {
	if v == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err == nil {
		return t
	}
	t, err = time.Parse(time.RFC3339, v)
	if err == nil {
		return t
	}
	return time.Time{}
}
