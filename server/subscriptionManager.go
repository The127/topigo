package server

import (
	"encoding/json"
	"github.com/The127/topigo/config"
	"github.com/The127/topigo/globals"
	"github.com/nanobox-io/golang-scribble"
	"log"
	"os"
	"path"
	"sync"
)

type subscriptionManager struct {
	subscriptions map[string]*subscription
	db            *scribble.Driver
	mu            sync.Mutex
}

func newSubscriptionManager(config config.Config) *subscriptionManager {
	dir := path.Join(config.Storage.Directory, "subscriptions")
	prepareDbFolder(config, dir)

	db, err := scribble.New(dir, nil)
	if err != nil {
		log.Fatalf("could not open subscription database: %v", err)
	}

	records, err := db.ReadAll("subscriptions")
	if err != nil {
		log.Printf("could not read subscriptions from database: %v", err)
	}

	subscriptions := make(map[string]*subscription)
	for _, f := range records {
		model := subscriptionModel{}
		if err := json.Unmarshal([]byte(f), &model); err != nil {
			log.Fatalf("could not unmarshal subscription record: %v", err)
		}
		subscription, err := newSubscription(&model)
		if err != nil {
			log.Fatalf("could not create subscription from model: %v", err)
		}
		subscriptions[model.Token] = subscription
	}

	return &subscriptionManager{
		db:            db,
		subscriptions: subscriptions,
	}
}

func prepareDbFolder(config config.Config, dir string) {
	if globals.VerboseLogging {
		log.Printf("initializing subscription db at (%v)", dir)
	}

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatalf("could not create storage folder 'subscription' in %v: %v", config.Storage.Directory, err)
	}
}

func (manager *subscriptionManager) createSubscriptionIfNotExists(request *CreateSubscriptionRequest) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, contains := manager.subscriptions[request.SubscriptionToken]; contains {
		if globals.VerboseLogging {
			log.Printf("subscription (%v) already exists", request.SubscriptionToken)
		}
		return false
	}

	if globals.VerboseLogging {
		log.Printf("creating subscription (%v)", request.SubscriptionToken)
	}

	model := subscriptionModel{
		Token: request.SubscriptionToken,
	}

	err := manager.db.Write("subscriptions", model.Token, model)
	if err != nil {
		log.Fatalf("could not write subscription (%v) to database: %v", model.Token, err)
	}

	subscription, err := newSubscription(&model)
	if err != nil {
		log.Fatalf("could not create subscription from model: %v", err)
	}
	manager.subscriptions[model.Token] = subscription
	return true
}

func (manager *subscriptionManager) modifySubscriptionIfExists(request *ModifySubscriptionRequest) ModifySubscriptionResponse_ModifySubscriptionResult {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	subscription, contains := manager.subscriptions[request.SubscriptionToken]
	if !contains {
		if globals.VerboseLogging {
			log.Printf("subscription (%v) does not exists", request.SubscriptionToken)
		}
		return ModifySubscriptionResponse_DoesNotExist
	}

	if subscription.Active() {
		if globals.VerboseLogging {
			log.Printf("subscription (%v) is already active", request.SubscriptionToken)
		}
		return ModifySubscriptionResponse_AlreadyInUse
	}

	if globals.VerboseLogging {
		log.Printf("modifying subscription (%v)", request.SubscriptionToken)
	}

	model := subscriptionModel{
		Token:  request.SubscriptionToken,
		Topics: request.Topics,
	}

	err := manager.db.Write("subscriptions", model.Token, model)
	if err != nil {
		log.Fatalf("could not update subscription (%v) in database: %v", model.Token, err)
	}

	subscription, err = newSubscription(&model)
	if err != nil {
		log.Fatalf("could not create subscription from model: %v", err)
	}
	manager.subscriptions[model.Token] = subscription
	return ModifySubscriptionResponse_Success
}

func (manager *subscriptionManager) deleteSubscriptionIfExists(subscriptionToken string) bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, contains := manager.subscriptions[subscriptionToken]; !contains {
		if globals.VerboseLogging {
			log.Printf("subscription (%v) does not exists", subscriptionToken)
		}
		return false
	}

	if globals.VerboseLogging {
		log.Printf("deleting subscription (%v)", subscriptionToken)
	}

	err := manager.db.Delete("subscriptions", subscriptionToken)
	if err != nil {
		log.Fatalf("could not delete subscription (%v) from database: %v", subscriptionToken, err)
	}

	delete(manager.subscriptions, subscriptionToken)
	return true
}

func (manager *subscriptionManager) GetSubscription(token string) (*subscription, bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	result, contains := manager.subscriptions[token]
	return result, contains
}
