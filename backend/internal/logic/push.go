package logic

import (
	"MessangerMax/internal/repo"
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type PushService struct {
	credentialsPath string
	tokenRepo       repo.DeviceTokenRepository
	httpClient      *http.Client

	mu          sync.Mutex
	tokenSource oauth2.TokenSource
	projectID   string
}

type fcmV1Message struct {
	Message fcmV1Payload `json:"message"`
}

type fcmV1Payload struct {
	Token        string            `json:"token"`
	Notification *fcmNotification  `json:"notification,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type FcmData struct {
	Type   string `json:"type"`
	ChatID string `json:"chat_id"`
}

type serviceAccountJSON struct {
	ProjectID string `json:"project_id"`
}

func NewPushService(credentialsPath string, tokenRepo repo.DeviceTokenRepository) *PushService {
	return &PushService{
		credentialsPath: credentialsPath,
		tokenRepo:       tokenRepo,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *PushService) SendPush(userID uint, title, body string, data *FcmData) {
	if s.credentialsPath == "" {
		return
	}

	tokens, err := s.tokenRepo.FindByUserID(userID)
	if err != nil {
		log.Printf("PushService: failed to get tokens for user %d: %v", userID, err)
		return
	}

	for _, t := range tokens {
		s.sendToToken(t.Token, title, body, data)
	}
}

func (s *PushService) sendToToken(token, title, body string, data *FcmData) {
	msg := fcmV1Message{
		Message: fcmV1Payload{
			Token: token,
			Notification: &fcmNotification{
				Title: title,
				Body:  body,
			},
		},
	}

	if data != nil {
		msg.Message.Data = map[string]string{
			"type":    data.Type,
			"chat_id": data.ChatID,
		}
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("PushService: marshal error: %v", err)
		return
	}

	project, err := s.getProjectID()
	if err != nil || project == "" {
		log.Printf("PushService: project ID unavailable: %v", err)
		return
	}

	url := "https://fcm.googleapis.com/v1/projects/" + project + "/messages:send"

	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		log.Printf("PushService: request error: %v", err)
		return
	}

	ts, err := s.getTokenSource()
	if err != nil {
		log.Printf("PushService: failed to get token source: %v", err)
		return
	}

	tok, err := ts.Token()
	if err != nil {
		log.Printf("PushService: failed to get OAuth2 token: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("PushService: send error to token %s: %v", maskToken(token), err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("PushService: FCM returned %d for token %s", resp.StatusCode, maskToken(token))
	}
}

func (s *PushService) getProjectID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.projectID != "" {
		return s.projectID, nil
	}

	data, err := os.ReadFile(s.credentialsPath)
	if err != nil {
		return "", err
	}

	var sa serviceAccountJSON
	if err := json.Unmarshal(data, &sa); err != nil {
		return "", err
	}

	s.projectID = sa.ProjectID
	return s.projectID, nil
}

func (s *PushService) getTokenSource() (oauth2.TokenSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tokenSource != nil {
		return s.tokenSource, nil
	}

	data, err := os.ReadFile(s.credentialsPath)
	if err != nil {
		return nil, err
	}

	scopes := []string{"https://www.googleapis.com/auth/firebase.messaging"}

	creds, err := google.CredentialsFromJSON(context.Background(), data, scopes...)
	if err != nil {
		return nil, err
	}

	s.tokenSource = creds.TokenSource
	return s.tokenSource, nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
