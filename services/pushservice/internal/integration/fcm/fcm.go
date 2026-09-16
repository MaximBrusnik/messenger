package fcm

import (
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

	"messengermax/pushservice/internal/entity"
	"messengermax/pushservice/internal/repo"
)

// Sender sends push notifications via FCM v1 HTTP API.
type Sender struct {
	credentialsPath string
	tokenRepo       repo.DeviceTokenRepository
	httpClient      *http.Client
	tokenSourceMu   sync.Mutex
	tokenSource     oauth2.TokenSource
	projectID       string
	projectIDMu     sync.Mutex
}

type fcmV1Message struct {
	Message fcmMessage `json:"message"`
}

type fcmMessage struct {
	Token        string            `json:"token"`
	Notification fcmNotification   `json:"notification"`
	Data         map[string]string `json:"data"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type serviceAccountJSON struct {
	ProjectID string `json:"project_id"`
}

func New(credentialsPath string, tokenRepo repo.DeviceTokenRepository) *Sender {
	return &Sender{
		credentialsPath: credentialsPath,
		tokenRepo:       tokenRepo,
		httpClient:      &http.Client{Timeout: 15 * time.Second},
	}
}

// SendPush delivers a title/body notification to all of the user's devices.
func (s *Sender) SendPush(ctx context.Context, userID uint, title, body string) {
	tokens, err := s.tokenRepo.FindByUserID(userID)
	if err != nil || len(tokens) == 0 {
		return
	}
	for i := range tokens {
		t := &tokens[i]
		go s.sendToToken(t, title, body)
	}
}

func (s *Sender) sendToToken(token *entity.DeviceToken, title, body string) {
	msg := fcmV1Message{
		Message: fcmMessage{
			Token:        token.Token,
			Notification: fcmNotification{Title: title, Body: body},
			Data:         map[string]string{"type": "new_message"},
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	bearer, err := s.getBearer()
	if err != nil {
		log.Printf("fcm: token error: %v", err)
		return
	}
	projectID, err := s.getProjectID()
	if err != nil {
		log.Printf("fcm: project id error: %v", err)
		return
	}
	url := "https://fcm.googleapis.com/v1/projects/" + projectID + "/messages:send"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("fcm: send failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("fcm: unexpected status %d for user %d", resp.StatusCode, token.UserID)
	}
}

func (s *Sender) getBearer() (string, error) {
	src, err := s.getTokenSource()
	if err != nil {
		return "", err
	}
	tok, err := src.Token()
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

func (s *Sender) getTokenSource() (oauth2.TokenSource, error) {
	if s.credentialsPath == "" {
		return nil, os.ErrNotExist
	}
	s.tokenSourceMu.Lock()
	defer s.tokenSourceMu.Unlock()
	if s.tokenSource != nil {
		return s.tokenSource, nil
	}
	data, err := os.ReadFile(s.credentialsPath)
	if err != nil {
		return nil, err
	}
	creds, err := google.CredentialsFromJSON(context.Background(), data, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return nil, err
	}
	s.tokenSource = creds.TokenSource
	return s.tokenSource, nil
}

func (s *Sender) getProjectID() (string, error) {
	if s.credentialsPath == "" {
		return "", os.ErrNotExist
	}
	s.projectIDMu.Lock()
	defer s.projectIDMu.Unlock()
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
