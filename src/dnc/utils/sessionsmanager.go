package utils

import "fmt"

type SessionManager struct {
	sessionsMap         RefCountedLockTable
	repositoryLockOrder []string
}

const sessionKey = "Repository_%s_Key_%s"

// NewSessionManager - Create session manager
func NewSessionManager(repoLockOrder []string) *SessionManager {
	sessionManager := new(SessionManager)
	sessionManager.repositoryLockOrder = repoLockOrder
	sessionManager.sessionsMap.Initialize()

	return sessionManager
}

// BeginReadSession - Begin read session
func (sessionManager *SessionManager) BeginReadSession(sessions map[string]string) {
	for _, repoName := range sessionManager.repositoryLockOrder {
		sessionID, ok := sessions[repoName]
		if ok {
			key := fmt.Sprintf(sessionKey, repoName, sessionID)
			sessionManager.sessionsMap.RLock(key)
		}
	}
}

// EndReadSession - End read session
func (sessionManager *SessionManager) EndReadSession(sessions map[string]string) {
	for _, repoName := range sessionManager.repositoryLockOrder {
		sessionID, ok := sessions[repoName]
		if ok {
			key := fmt.Sprintf(sessionKey, repoName, sessionID)
			sessionManager.sessionsMap.RUnlock(key)
		}
	}
}

// BeginWriteSession - Begin write session
func (sessionManager *SessionManager) BeginWriteSession(sessions map[string]string) {
	for _, repoName := range sessionManager.repositoryLockOrder {
		sessionID, ok := sessions[repoName]
		if ok {
			key := fmt.Sprintf(sessionKey, repoName, sessionID)
			sessionManager.sessionsMap.Lock(key)
		}
	}
}

// EndWriteSession - End write session
func (sessionManager *SessionManager) EndWriteSession(sessions map[string]string) {
	for _, repoName := range sessionManager.repositoryLockOrder {
		sessionID, ok := sessions[repoName]
		if ok {
			key := fmt.Sprintf(sessionKey, repoName, sessionID)
			sessionManager.sessionsMap.Unlock(key)
		}
	}
}
