package webhook

import (
	"encoding/json"
	"sync"
	"time"

	"lucor.dev/beebuzz/internal/secure"
)

const inspectSessionTTL = 10 * time.Minute

type InspectStatus string

const (
	InspectStatusWaiting   InspectStatus = "waiting"
	InspectStatusCaptured  InspectStatus = "captured"
	InspectStatusCompleted InspectStatus = "completed"
	InspectStatusExpired   InspectStatus = "expired"
)

type InspectSession struct {
	UserID      string
	TokenHash   string
	Name        string
	Description string
	Priority    string
	Topics      []string
	Status      InspectStatus
	Payload     json.RawMessage
	ExpiresAt   time.Time
	CapturedAt  *time.Time
}

type InspectStore struct {
	sessions map[string]*InspectSession // keyed by userID
	mu       sync.Mutex
}

// NewInspectStore returns a new in-memory inspect session store.
func NewInspectStore() *InspectStore {
	return &InspectStore{
		sessions: make(map[string]*InspectSession),
	}
}

// Create generates a new inspect session for the given user, replacing any previous one.
func (s *InspectStore) Create(userID, name, description, priority string, topics []string) (string, *InspectSession, error) {
	rawToken, err := secure.NewInspectToken()
	if err != nil {
		return "", nil, err
	}

	tokenHash := secure.Hash(rawToken)

	session := &InspectSession{
		UserID:      userID,
		TokenHash:   tokenHash,
		Name:        name,
		Description: description,
		Priority:    priority,
		Topics:      topics,
		Status:      InspectStatusWaiting,
		ExpiresAt:   time.Now().Add(inspectSessionTTL),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()
	s.sessions[userID] = session

	return rawToken, session, nil
}

// GetByUserID returns the inspect session for the given user, or nil if not found or expired.
func (s *InspectStore) GetByUserID(userID string) *InspectSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[userID]
	if !ok {
		return nil
	}

	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, userID)
		return nil
	}

	return session
}

// GetByTokenHash returns the inspect session matching the given token hash, or nil if not found or expired.
func (s *InspectStore) GetByTokenHash(tokenHash string) *InspectSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	for userID, session := range s.sessions {
		if session.TokenHash != tokenHash {
			continue
		}
		if time.Now().After(session.ExpiresAt) {
			delete(s.sessions, userID)
			return nil
		}
		return session
	}

	return nil
}

// Capture records the payload for the inspect session identified by token hash.
func (s *InspectStore) Capture(tokenHash string, payload json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, session := range s.sessions {
		if session.TokenHash != tokenHash {
			continue
		}
		if time.Now().After(session.ExpiresAt) {
			delete(s.sessions, session.UserID)
			return ErrInspectSessionNotFound
		}
		if session.Status != InspectStatusWaiting {
			return ErrInspectNotWaiting
		}
		now := time.Now()
		session.Payload = payload
		session.Status = InspectStatusCaptured
		session.CapturedAt = &now
		return nil
	}

	return ErrInspectSessionNotFound
}

// Delete removes the inspect session for the given user.
func (s *InspectStore) Delete(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, userID)
}

// cleanupLocked removes all expired sessions. Must be called with mu held.
func (s *InspectStore) cleanupLocked() {
	now := time.Now()
	for userID, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, userID)
		}
	}
}
