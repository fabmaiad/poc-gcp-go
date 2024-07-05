package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/sirupsen/logrus"
)

type Message struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type PubsubService struct {
	projectId string
	client    *pubsub.Client
}

func NewPubsubService() (*PubsubService, error) {
	logrus.Debug("Starting NewPubsubService")
	projectId := os.Getenv("GCP_PROJECT_ID")
	logrus.Debugf("GCP_PROJECT_ID: %s", projectId)

	client, err := pubsub.NewClient(context.Background(), projectId)
	if err != nil {
		logrus.Errorf("Failed to create PubSub client: %v", err)
		return nil, fmt.Errorf("pubsub: NewPubsubService: %w", err)
	}

	logrus.Debug("PubSub client created successfully")
	return &PubsubService{
		projectId: projectId,
		client:    client,
	}, nil
}

func (s *PubsubService) PublisherMessage(ctx context.Context, topicID, message string) (string, error) {
	logrus.Debug("Starting PublisherMessage")
	topic := s.client.Topic(topicID)
	defer func() {
		logrus.Debug("Stoping topic")
		topic.Stop()
	}()
	logrus.Debug("Topic Ref obtained")

	result := topic.Publish(ctx, &pubsub.Message{
		Data: []byte(message),
	})
	logrus.Debug("Message published, waiting for result")

	id, err := result.Get(ctx)
	if err != nil {
		logrus.Errorf("Failed to get result: %v", err)
		return "", fmt.Errorf("FALHA: %w", err)
	}

	logrus.Debug("Message published successfully")
	msg := fmt.Sprintf("MSG ID: %s", id)
	logrus.Debug(msg)

	return msg, nil
}

func init() {
	functions.HTTP("Main", main)
}

func checkNetworkConnectivity() error {
	pubsubServiceHost := "pubsub.googleapis.com:443"
	conn, err := net.DialTimeout("tcp", pubsubServiceHost, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func main(w http.ResponseWriter, r *http.Request) {
	logrus.SetLevel(logrus.DebugLevel)
	// URL do endpoint
	url := os.Getenv("ENDPOINT_SERVER")
	logrus.Debugf("Fetching URL: %s", url)

	// Fazendo a requisição GET
	resp, err := http.Get(url)
	if err != nil {
		logrus.Fatalf("Erro ao fazer a requisição: %v", err)
	}
	defer resp.Body.Close()

	// Lendo o corpo da resposta
	logrus.Debug("Request successful, reading body")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Fatalf("Erro ao ler o corpo da resposta: %v", err)
	}

	// Parse do JSON recebido
	var messages []Message
	if err := json.Unmarshal(body, &messages); err != nil {
		logrus.Fatalf("Erro ao fazer parse do JSON: %v", err)
	}

	if err := checkNetworkConnectivity(); err != nil {
		logrus.Fatalf("Network connectivity test failed: %v", err)
	} else {
		logrus.Debug("Network connectivity test passed")
	}

	service, err := NewPubsubService()
	if err != nil {
		logrus.Fatalf("Error creating service: %v", err)
		return
	}

	topicID := os.Getenv("TOPIC_ID")
	logrus.Debugf("Publishing messages to topic: %s", topicID)

	// Cria o Context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	// Publicando cada mensagem individualmente
	for _, msg := range messages {
		messageJSON, err := json.Marshal(msg)
		if err != nil {
			logrus.Errorf("Erro ao converter mensagem para JSON: %v", err)
			continue
		}

		logrus.Debug("Publishing message")
		_, err = service.PublisherMessage(ctx, topicID, string(messageJSON))
		if err != nil {
			logrus.Errorf("Error publishing message: %v", err)
			return
		}
	}
	logrus.Debug("All messages published successfully")
}
