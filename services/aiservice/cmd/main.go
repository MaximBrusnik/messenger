package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"messengermax/aiservice/internal/integration/gemini"
	"messengermax/pkg/config"
	"messengermax/pkg/grpcsrv"
	"messengermax/pkg/nats"
	pbchat "messengermax/proto/gen/chat"
)

const (
	historyLimit = 20
	maxTryReads  = 3
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "aiservice"

	geminiClient := gemini.NewClient(cfg.AIConfig.GeminiAPIKey)
	if geminiClient == nil {
		log.Println("ai: GEMINI_API_KEY not set, AI assistant disabled")
	}

	chatConn, err := grpcsrv.Dial(cfg.Services.ChatAddr)
	if err != nil {
		log.Fatal("ai: chatservice unreachable: ", err)
	}
	defer chatConn.Close()
	chatClient := pbchat.NewChatServiceClient(chatConn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handle := func(topic, key string, value []byte) {
		handleTrigger(ctx, chatClient, geminiClient, value)
	}
	consumer := nats.NewConsumer(cfg.NATS.URL, "aiservice", []string{nats.TopicAITrigger}, handle)
	go consumer.Run(ctx)

	log.Println("ai: listening for AI triggers")
	select {}
}

func handleTrigger(ctx context.Context, chatClient pbchat.ChatServiceClient, geminiClient *gemini.Client, value []byte) {
	if geminiClient == nil {
		return
	}
	var e nats.EventMessageCreated
	if err := json.Unmarshal(value, &e); err != nil {
		return
	}
	botID := uint64(e.BotID)
	if botID == 0 {
		return
	}

	// brief backoff so the user message is readable before the reply
	time.Sleep(500 * time.Millisecond)

	history := make([]gemini.ChatMessage, 0, historyLimit)
	// retry a few times because messages may still be in flight to chatservice
	for attempt := 0; attempt < maxTryReads; attempt++ {
		resp, err := chatClient.GetMessages(ctx, &pbchat.GetMessagesRequest{
			UserId: botID,
			ChatId: uint64(e.ChatID),
			Limit:  historyLimit,
		})
		if err == nil && len(resp.Messages) > 0 {
			for _, m := range resp.Messages {
				if m.Text == "" {
					continue
				}
				history = append(history, gemini.ChatMessage{
					Text:  m.Text,
					IsBot: m.SenderId == botID,
				})
			}
			break
		}
		if attempt < maxTryReads-1 {
			time.Sleep(700 * time.Millisecond)
		}
	}
	if len(history) == 0 {
		return
	}

	answer, err := geminiClient.GenerateResponse(history)
	if err != nil {
		log.Printf("ai: generate failed: %v", err)
		return
	}
	if answer == "" {
		return
	}

	// post the assistant reply back into the same chat
	_, err = chatClient.SendMessage(ctx, &pbchat.SendMessageRequest{
		UserId:  botID,
		ChatId:  uint64(e.ChatID),
		Content: answer,
	})
	if err != nil {
		log.Printf("ai: failed to post reply: %v", err)
	}
}
