package server

import (
	"regexp"
	"sync"
)

type subscriptionModel struct {
	Token  string   `json:"token"`
	Topics []string `json:"topics"`
}

type subscription struct {
	token        string
	topicRegexps []*regexp.Regexp
	active       bool
	closeSignal  chan bool
	messages     chan *messageDto
	mu           sync.Mutex
}

func newSubscription(model *subscriptionModel) (*subscription, error) {
	var regexps []*regexp.Regexp

	for _, topic := range model.Topics {
		compiled, err := regexp.Compile(topic)
		if err != nil {
			return nil, err
		}

		regexps = append(regexps, compiled)
	}

	return &subscription{
		token:        model.Token,
		topicRegexps: regexps,
		active:       false,
		closeSignal:  make(chan bool),
		messages:     make(chan *messageDto),
	}, nil
}

func (subscription *subscription) Active() bool {
	return subscription.active
}

func (subscription *subscription) StartStreaming() bool {
	subscription.mu.Lock()
	defer subscription.mu.Unlock()

	if subscription.active {
		return false
	}

	subscription.active = true
	return true
}

func (subscription *subscription) StopStreaming() bool {
	subscription.mu.Lock()
	subscription.mu.Unlock()

	if !subscription.active {
		return false
	}

	subscription.active = false
	subscription.closeSignal <- true
	return true
}

func (subscription *subscription) IsSubscribed(topic string) bool {
	for _, compiledRegex := range subscription.topicRegexps {
		if compiledRegex.MatchString(topic) {
			return true
		}
	}
	return false
}

func (subscription *subscription) HandleStreamError() {
	subscription.mu.Lock()
	subscription.mu.Unlock()
	subscription.active = false
}
