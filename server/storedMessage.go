package server

import (
	"github.com/rs/xid"
	"time"
)

type storedMessage struct {
	Name       string    `json:"name"`
	Topic      string    `json:"topic"`
	From       string    `json:"from"`
	ReceivedBy []string  `json:"receivedBy"`
	Received   time.Time `json:"received"`
	Content    string    `json:"content"`
}

func makeStoredMessage(topic, from, content string, received time.Time) storedMessage {
	return storedMessage{Content: content, Name: xid.New().String(), Topic: topic, From: from, ReceivedBy: nil, Received: received}
}
