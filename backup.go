package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sync"
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
	runtime.GOMAXPROCS(1)
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

	var wg sync.WaitGroup

	for _, msg := range messages {
		wg.Add(1)
		go func(msg Message) {
			defer wg.Done()

			messageJSON, err := json.Marshal(msg)
			if err != nil {
				logrus.Errorf("Erro ao converter mensagem para JSON: %v", err)
				return
			}

			m := &pubsub.Message{
				Data: []byte(messageJSON),
			}

			startTime := time.Now()
			id, err := client.Topic(topicID).Publish(ctx, m).Get(ctx)
			duration := time.Since(startTime)

			if err != nil {
				logrus.Errorf("topic(%s).Publish.Get (durou %v): %v", topicID, duration, err)
				http.Error(w, fmt.Sprintf("Erro ao publicar a mensagem: %v", err), http.StatusInternalServerError)
				return
			}
			logrus.Infof("Mensagem publicada (durou %v): %v", duration, id)
			fmt.Fprintf(w, "Mensagem publicada: %v\n", id)
		}(msg)
	}
	wg.Wait()
	logrus.Debug("ALL Messages Published sucessfully")
}
