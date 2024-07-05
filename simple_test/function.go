package main

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub"
)

func main() {
	projectID := "bullla-one-d-apps-cn-92c3"
	topicID := "topic1-poc-golang"
	message := "Hello, World!"

	err := publishMessage(projectID, topicID, message)
	if err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}
}

func publishMessage(projectID, topicID, message string) error {
	ctx := context.Background()

	// Create a new Pub/Sub client
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("pubsub.NewClient: %v", err)
	}
	defer client.Close()

	// Get the topic
	topic := client.Topic(topicID)

	// Publish the message
	result := topic.Publish(ctx, &pubsub.Message{
		Data: []byte(message),
	})

	// Block until the result is returned and a server-generated
	// ID is returned for the published message.
	id, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("Get: %v", err)
	}
	fmt.Printf("Published a message; msg ID: %v\n", id)
	return nil
}
