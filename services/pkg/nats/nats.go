package nats

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	streamName = "MESSENGER"
	keyHeader  = "X-Message-Key"

	TopicCallEnded = "calls.call.ended"

	TopicUserEvents      = "user.events"
	TopicMessageCreated  = "chat.message.created"
	TopicMessageEdited   = "chat.message.edited"
	TopicMessageDeleted  = "chat.message.deleted"
	TopicMessagePinned   = "chat.message.pinned"
	TopicMessageUnpinned = "chat.message.unpinned"
	TopicReactionAdded   = "chat.reaction.added"
	TopicReactionRemoved = "chat.reaction.removed"
	TopicReadReceived    = "chat.read"
	TopicChatDeleted     = "chat.deleted"
	TopicAITrigger       = "chat.ai.trigger"
)

var AllTopics = []string{
	TopicCallEnded,
	TopicUserEvents,
	TopicMessageCreated,
	TopicMessageEdited,
	TopicMessageDeleted,
	TopicMessagePinned,
	TopicMessageUnpinned,
	TopicReactionAdded,
	TopicReactionRemoved,
	TopicReadReceived,
	TopicChatDeleted,
	TopicAITrigger,
}

func connect(url string) (*nats.Conn, nats.JetStreamContext) {
	var nc *nats.Conn
	var err error
	for i := 0; i < 30; i++ {
		nc, err = nats.Connect(url,
			nats.MaxReconnects(100),
			nats.ReconnectWait(500*time.Millisecond),
		)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil || nc == nil {
		log.Printf("nats: connect %s failed: %v", url, err)
		return nil, nil
	}
	js, err := nc.JetStream()
	if err != nil {
		log.Printf("nats: jetstream unavailable: %v", err)
		return nc, nil
	}
	if err := ensureStream(js); err != nil {
		log.Printf("nats: stream setup failed: %v", err)
	}
	return nc, js
}

func ensureStream(js nats.JetStreamContext) error {
	cfg := &nats.StreamConfig{
		Name:      streamName,
		Subjects:  AllTopics,
		Retention: nats.LimitsPolicy,
		Storage:   nats.FileStorage,
		MaxAge:    24 * time.Hour,
	}
	info, err := js.StreamInfo(streamName)
	if err != nil {
		_, err = js.AddStream(cfg)
		return err
	}
	// Merge any newly added topics into the existing stream config so the
	// stream keeps capturing them after startup.
	existing := info.Config.Subjects
	seen := make(map[string]bool, len(existing))
	for _, s := range existing {
		seen[s] = true
	}
	merged := existing
	for _, s := range AllTopics {
		if !seen[s] {
			seen[s] = true
			merged = append(merged, s)
		}
	}
	if len(merged) != len(existing) {
		info.Config.Subjects = merged
		_, err = js.UpdateStream(&info.Config)
	}
	return err
}

type Producer struct {
	mu  sync.Mutex
	url string
	nc  *nats.Conn
	js  nats.JetStreamContext
}

func NewProducer(url string) *Producer {
	p := &Producer{url: url}
	p.connect()
	return p
}

func (p *Producer) connect() {
	nc, js := connect(p.url)
	p.mu.Lock()
	p.nc, p.js = nc, js
	p.mu.Unlock()
}

func (p *Producer) Publish(topic string, key string, value interface{}) {
	p.mu.Lock()
	js := p.js
	p.mu.Unlock()
	if js == nil {
		log.Printf("nats: not connected, dropping %s", topic)
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("nats: marshal %s failed: %v", topic, err)
		return
	}
	hdr := nats.Header{}
	if key != "" {
		hdr.Set(keyHeader, key)
	}
	msg := &nats.Msg{Subject: topic, Header: hdr, Data: data}
	if _, err := js.PublishMsg(msg); err != nil {
		log.Printf("nats: publish %s failed: %v", topic, err)
	}
}

func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.nc != nil {
		p.nc.Close()
	}
	return nil
}

type Consumer struct {
	nc     *nats.Conn
	subs   []*nats.Subscription
	handle func(topic string, key string, value []byte)
}

func NewConsumer(url string, groupID string, topics []string, handle func(topic, key string, value []byte)) *Consumer {
	c := &Consumer{handle: handle}
	nc, js := connect(url)
	if js == nil {
		return c
	}
	c.nc = nc
	for _, topic := range topics {
		durable := groupID + "-" + strings.ReplaceAll(topic, ".", "-")
		sub, err := js.QueueSubscribe(topic, groupID, func(m *nats.Msg) {
			if c.handle != nil {
				c.handle(m.Subject, m.Header.Get(keyHeader), m.Data)
			}
			_ = m.Ack()
		},
			nats.Durable(durable),
			nats.ManualAck(),
			nats.AckExplicit(),
			nats.AckWait(5*time.Minute),
		)
		if err != nil {
			log.Printf("nats: subscribe %s failed: %v", topic, err)
			continue
		}
		c.subs = append(c.subs, sub)
	}
	return c
}

func (c *Consumer) Run(ctx context.Context) {
	<-ctx.Done()
	for _, s := range c.subs {
		_ = s.Unsubscribe()
	}
	if c.nc != nil {
		_ = c.nc.Drain()
	}
}
