package main

import (
	"log"
	"os"

	// Blank-import the function package so the init() runs
	_ "example.com/hello"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/sirupsen/logrus"
)

func main() {
	// Use PORT environment variable, or default to 8080.
	port := "8081"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	// By default, listen on all interfaces. If testing locally, run with
	// LOCAL_ONLY=true to avoid triggering firewall warnings and
	// exposing the server outside of your own machine.
	hostname := ""
	if localOnly := os.Getenv("LOCAL_ONLY"); localOnly == "true" {
		hostname = "127.0.0.1"
	}
	logrus.Infof("Starting server on %s:%s", hostname, port)
	if err := funcframework.StartHostPort(hostname, port); err != nil {
		log.Fatalf("funcframework.StartHostPort: %v\n", err)
	}
}
