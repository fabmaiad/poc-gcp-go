package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/sirupsen/logrus"
)

// Client Global PubSub
var client *pubsub.Client
var once sync.Once

// CreateClient
func createClient() {
	var projectID = os.Getenv("GCP_PROJECT_ID")
	var err error

	client, err = pubsub.NewClient(context.Background(), projectID)
	if err != nil {
		logrus.Fatalf("pubsub.NewClient: %v", err)
	}
}

func init() {
	//
	//runtime.GOMAXPROCS(2)
	// Registrando HTTP Function
	functions.HTTP("Main", PublishMessage)
}

type Message struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

func fetchMessages() ([]Message, error) {
	// URL do endpoint
	url := os.Getenv("ENDPOINT_SERVER")
	if url == "" {
		return nil, fmt.Errorf("ENDPOINT_SERVER is not set")
	}
	logrus.Debugf("Fetching URL: %s", url)

	// Fazendo a requisição GET
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer a requisição: %w", err)
	}
	defer resp.Body.Close()

	// Lendo o corpo da resposta
	logrus.Debug("Request successful, reading body")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o corpo da resposta: %w", err)
	}

	// Parse do JSON recebido
	var messages []Message
	if err := json.Unmarshal(body, &messages); err != nil {
		return nil, fmt.Errorf("erro ao fazer parse do JSON: %w", err)
	}

	return messages, nil
}

func PublishMessage(w http.ResponseWriter, r *http.Request) {
	logrus.SetLevel(logrus.DebugLevel)

	var messages []Message
	var topicID string = os.Getenv("TOPIC_ID")
	if topicID == "" {
		http.Error(w, "TOPIC_ID is not set", http.StatusInternalServerError)
		return
	}

	once.Do(createClient)

	messages, err := fetchMessages()
	if err != nil {
		logrus.Fatalf("Falha ao recuperar mensagens: %v", err)
		http.Error(w, fmt.Sprintf("Falha ao recuperar mensagens: %v", err), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Minute)
	defer cancel()

	t := client.Topic(topicID)
	t.PublishSettings.FlowControlSettings = pubsub.FlowControlSettings{
		MaxOutstandingMessages: 100,
		MaxOutstandingBytes:    10 * 1024 * 1024,
		LimitExceededBehavior:  pubsub.FlowControlBlock,
	}

	var wg sync.WaitGroup
	var totalErrors uint64

	numMsgs := len(messages)
	for i, msg := range messages {
		wg.Add(1)
		messageJSON, err := json.Marshal(msg)
		if err != nil {
			logrus.Errorf("Erro ao converter mensagem para JSON: %v", err)
			wg.Done()
			atomic.AddUint64(&totalErrors, 1)
			continue
		}

		result := t.Publish(ctx, &pubsub.Message{
			Data: []byte(messageJSON),
		})

		go func(i int, res *pubsub.PublishResult) {
			defer wg.Done()
			_, err := res.Get(ctx)
			if err != nil {
				logrus.Errorf("Failed to publish message %d: %v", i, err)
				atomic.AddUint64(&totalErrors, 1)
				return
			}
			logrus.Infof("Successfully published message %d", i)
		}(i, result)
	}

	wg.Wait()

	if totalErrors > 0 {
		http.Error(w, fmt.Sprintf("%d of %d messages did not publish successfully", totalErrors, numMsgs), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "All messages published successfully")
	logrus.Debug("All messages published successfully")
}
