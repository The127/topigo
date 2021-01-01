package server

import (
	"context"
	"encoding/json"
	scribble "github.com/nanobox-io/golang-scribble"
	"log"
	"os"
	"path"
	"sort"
	"sync"
	"time"
	"topigo/src/config"
	"topigo/src/globals"
)

type topicgoServer struct {
	UnimplementedTopigoServer
	subscriptionManager *subscriptionManager
	db                  *scribble.Driver
}

func MakeTopicoServer(config config.Config) topicgoServer {
	messageDbPath := path.Join(config.Storage.Directory, "messages")
	err := os.MkdirAll(messageDbPath, os.ModePerm)
	if err != nil {
		log.Fatalf("could not create storage folder 'messages' in %v: %v", config.Storage.Directory, err)
	}

	db, err := scribble.New(messageDbPath, nil)
	if err != nil {
		log.Fatalf("could not create message database: %v", err)
	}

	cleanTicker := time.NewTicker(time.Second * time.Duration(config.Storage.DeletionIntervalHours))
	go ScheduleMessageCleaner(db, cleanTicker, config)

	//retryTicker := time.NewTicker(time.Second * time.Duration(config.Storage.DeletionIntervalHours))
	//go RetryScheduler(db, retryTicker, config)

	manager := newSubscriptionManager(config)

	messages = make(chan storedMessage)
	go DeliverMessages(db, manager)

	return topicgoServer{
		subscriptionManager: manager,
		db:                  db,
	}
}

func ScheduleMessageCleaner(db *scribble.Driver, ticker *time.Ticker, config config.Config) {
	mu := sync.Mutex{}
	isCleaning := false

	for _ = range ticker.C {
		if globals.VerboseLogging {
			log.Printf("trying to clean messages")
		}
		if !CleanMessages(&mu, &isCleaning, db, config) {
			if globals.VerboseLogging {
				log.Printf("message cleaning still in progress")
			}
			continue
		}
	}
}

func CleanMessages(mu *sync.Mutex, isCleaning *bool, db *scribble.Driver, config config.Config) bool {
	mu.Lock()
	if *isCleaning {
		mu.Unlock()
		return false
	} else {
		*isCleaning = true
		mu.Unlock()
	}

	log.Println("starting message cleanup")
	cleanupCount := 0

	records, err := db.ReadAll("messages")
	if err != nil {
		log.Printf("error while cleaning messages: could not read messages from database: %v", err)
	}

	now := time.Now()

	for _, record := range records {
		message := storedMessage{}
		if err := json.Unmarshal([]byte(record), &message); err != nil {
			log.Fatalf("could not create message from record: %v", err)
		}
		deletionInterval := time.Duration(config.Storage.RetentionDays) * time.Hour * 24
		deletionBuffer := time.Duration(config.Storage.DeletionBufferDays) * time.Hour * 24
		if message.Received.Add(deletionInterval).Add(deletionBuffer).Before(now) {
			if err := db.Delete("messages", message.Name); err != nil {
				log.Fatalf("could not delete message (%v): %v", message.Name, err)
			}
			cleanupCount++
		}
	}

	log.Printf("cleaned up %v messages", cleanupCount)

	mu.Lock()
	*isCleaning = false
	mu.Unlock()

	return true
}

func (server *topicgoServer) CreateSubscription(ctx context.Context, request *CreateSubscriptionRequest) (*CreateSubscriptionResponse, error) {
	result := CreateSubscriptionResponse_Exists
	if server.subscriptionManager.createSubscriptionIfNotExists(request) {
		result = CreateSubscriptionResponse_Created
	}

	return &CreateSubscriptionResponse{
		Result: result,
	}, nil
}

func (server *topicgoServer) ModifySubscription(ctx context.Context, request *ModifySubscriptionRequest) (*ModifySubscriptionResponse, error) {
	return &ModifySubscriptionResponse{
		Result: server.subscriptionManager.modifySubscriptionIfExists(request),
	}, nil
}

func (server *topicgoServer) DeleteSubscription(ctx context.Context, request *DeleteSubscriptionRequest) (*DeleteSubscriptionResponse, error) {
	result := DeleteSubscriptionResponse_Error
	if server.subscriptionManager.deleteSubscriptionIfExists(request.SubscriptionToken) {
		result = DeleteSubscriptionResponse_Deleted
	}

	return &DeleteSubscriptionResponse{
		Result: result,
	}, nil
}

func (server *topicgoServer) StartSubscriptionStreaming(request *StartSubscriptionStreamingRequest, stream Topigo_StartSubscriptionStreamingServer) error {
	subscription, contains := server.subscriptionManager.GetSubscription(request.SubscriptionToken)
	if !contains {
		return &SubscriptionDoesNotExistError{}
	}
	if !subscription.StartStreaming() {
		return &SubscriptionAlreadyStreamingError{}
	}

	go HandleMissedMessages(subscription, server.db)

	for {
		select {
		case <-subscription.closeSignal:
			return nil
		case message := <-subscription.messages:
			if subscription.token == message.from {
				continue
			}

			if subscription.IsSubscribed(message.topic) {
				if err := stream.Send(&Message{
					Topic:   message.topic,
					Content: message.content,
				}); err != nil {
					subscription.HandleStreamError()
					return err
				}
			}
		case <-stream.Context().Done():
			subscription.HandleStreamError()
			return nil
		}
	}
}

func HandleMissedMessages(sub *subscription, db *scribble.Driver) {
	records, err := db.ReadAll("messages")
	if err != nil {
		log.Fatalf("could not read messages from database: %v", err)
	}
	var messages []*storedMessage
	for _, record := range records {
		message := storedMessage{}
		if err := json.Unmarshal([]byte(record), &message); err != nil {
			log.Fatalf("could not unmarshal message: %v", err)
		}

		if NeedsToBeSentToSubscription(sub.token, message.ReceivedBy) {
			messages = append(messages, &message)
		}
	}

	sort.Slice(messages, func(p, q int) bool {
		return messages[p].Received.Before(messages[q].Received)
	})

	for _, message := range messages {
		sub.messages <- &messageDto{
			from:    message.From,
			topic:   message.Topic,
			content: message.Content,
		}
		message.ReceivedBy = append(message.ReceivedBy, sub.token)
		db.Write("messages", message.Name, message)
	}
}

func NeedsToBeSentToSubscription(subscriptionToken string, receivers []string) bool {
	for _, receiver := range receivers {
		if receiver == subscriptionToken {
			return false
		}
	}
	return true
}

func (server *topicgoServer) EndSubscriptionStreaming(ctx context.Context, request *EndSubscriptionStreamingRequest) (*EndSubscriptionStreamingResponse, error) {
	subscription, contains := server.subscriptionManager.GetSubscription(request.SubscriptionToken)
	if !contains {
		return nil, &SubscriptionDoesNotExistError{}
	}
	if !subscription.StopStreaming() {
		return &EndSubscriptionStreamingResponse{
			Result: EndSubscriptionStreamingResponse_Error,
		}, nil
	} else {
		return &EndSubscriptionStreamingResponse{
			Result: EndSubscriptionStreamingResponse_Ended,
		}, nil
	}
}

func (server *topicgoServer) Publish(ctx context.Context, request *PublishRequest) (*PublishResponse, error) {
	storedMessage := makeStoredMessage(request.Message.Topic, request.SubscriptionToken, request.Message.Content, time.Now())
	if err := server.db.Write("messages", storedMessage.Name, storedMessage); err != nil {
		return nil, err
	}
	go func() { messages <- storedMessage }()
	return &PublishResponse{}, nil
}

var messages chan storedMessage

func DeliverMessages(db *scribble.Driver, manager *subscriptionManager) {
	for {
		message := <-messages
		dto := messageDto{
			from:    message.From,
			topic:   message.Topic,
			content: message.Content,
		}
		for _, sub := range manager.subscriptions {
			if sub.token != message.From && sub.active {
				select {
				case sub.messages <- &dto:
					message.ReceivedBy = append(message.ReceivedBy, sub.token)
				default:
					log.Printf("subscriber closed: %v\n", sub.token)
				}
			}
		}
		db.Write("messages", message.Name, message)
	}
}
