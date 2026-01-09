//go:build !tdlib

package telegram

import "fmt"

// stubImpl is a stub implementation that doesn't require TDLib
// Used for development and testing without TDLib installed
type stubImpl struct {
	client *Client
}

func newClientImpl(c *Client) (clientImpl, error) {
	return &stubImpl{client: c}, nil
}

func (s *stubImpl) start() error {
	// Simulate connecting and waiting for phone number
	s.client.setConnectionState(ConnectionStateConnecting)
	s.client.setAuthState(AuthStateWaitPhoneNumber)
	return nil
}

func (s *stubImpl) close() error {
	return nil
}

func (s *stubImpl) sendPhoneNumber(phone string) error {
	// Simulate success - transition to waiting for code
	s.client.setAuthState(AuthStateWaitCode)
	return nil
}

func (s *stubImpl) sendAuthCode(code string) error {
	// Simulate success - transition to ready
	s.client.setAuthState(AuthStateReady)
	s.client.setConnectionState(ConnectionStateReady)
	return nil
}

func (s *stubImpl) send2FAPassword(password string) error {
	// Simulate success - transition to ready
	s.client.setAuthState(AuthStateReady)
	s.client.setConnectionState(ConnectionStateReady)
	return nil
}

func (s *stubImpl) getChats(limit int) ([]*Chat, error) {
	return nil, fmt.Errorf("stub: TDLib not available")
}

func (s *stubImpl) getChat(chatID int64) (*Chat, error) {
	return nil, fmt.Errorf("stub: TDLib not available")
}

func (s *stubImpl) searchPublicChat(username string) (*Chat, error) {
	return nil, fmt.Errorf("stub: TDLib not available")
}

func (s *stubImpl) getChatHistory(chatID int64, fromMessageID int64, limit int) ([]*Message, error) {
	return nil, fmt.Errorf("stub: TDLib not available")
}

func (s *stubImpl) sendMessage(chatID int64, text string, replyToID int64) (*Message, error) {
	return nil, fmt.Errorf("stub: TDLib not available")
}

func (s *stubImpl) markAsRead(chatID int64, messageIDs []int64) error {
	return nil // Silently succeed
}

func (s *stubImpl) downloadFile(fileID string, priority int) error {
	return fmt.Errorf("stub: TDLib not available")
}

func (s *stubImpl) cancelDownload(fileID string) error {
	return nil // Silently succeed
}
