package main

import (
	"context"
	"log"

	"google.golang.org/cloud/datastore"
	"google.golang.org/cloud/pubsub"
)

func main() {
	ctx := context.Background()

	pubsubClient, err := pubsub.NewClient(ctx, "project")
	if err != nil {
		log.Fatalf("pubsub.NewClient: %v", err)
	}
	topic, err := pubsubClient.NewTopic(ctx, "topic")
	if err != nil {
		log.Fatalf("NewTopic: %v", err)
	}
	sub, err := pubsubClient.NewSubscription(ctx, "sub", topic, 0, nil)
	if err != nil {
		log.Fatalf("NewSubscription: %v", err)
	}
	if _, err := topic.Publish(ctx, &pubsub.Message{
		Data: []byte("hello"),
	}); err != nil {
		log.Fatalf("Publish: %v", err)
	}
	it, err := sub.Pull(ctx)
	if err != nil {
		log.Fatalf("Pull: %v", err)
	}
	msg, err := it.Next()
	if err != nil {
		log.Fatalf("Next: %v", err)
	}
	log.Printf("Message: %s", msg.Data)
	msg.Done(true)
	it.Stop()

	datastoreClient, err := datastore.NewClient(ctx, "project")
	if err != nil {
		log.Fatalf("datastore.NewClient: %v", err)
	}
	k := datastore.NewIncompleteKey(ctx, "Foo", nil)
	if _, err := datastoreClient.Put(ctx, k, &Foo{F: "foo!"}); err != nil {
		log.Fatalf("Put: %v", err)
	}
}

type Foo struct {
	F string
}
