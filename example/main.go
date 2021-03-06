// Copyright 2016 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"log"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
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
	log.Printf("pubsub message: %s", msg.Data)
	msg.Done(true)
	it.Stop()

	datastoreClient, err := datastore.NewClient(ctx, "project")
	if err != nil {
		log.Fatalf("datastore.NewClient: %v", err)
	}
	k := datastore.NewIncompleteKey(ctx, "Foo", nil)
	k, err = datastoreClient.Put(ctx, k, &Foo{F: "foo!"})
	if err != nil {
		log.Fatalf("Put: %v", err)
	}

	var f Foo
	if err := datastoreClient.Get(ctx, k, &f); err != nil {
		log.Fatalf("Get: %v", err)
	}
	log.Printf("datastore got %v", f)
}

type Foo struct {
	F string
}
