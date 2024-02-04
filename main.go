package main

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/logutils"
	"github.com/slack-go/slack"
)

const (
	defaultListenAddr   = "127.0.0.1:8080"
	defaultRedisDB      = 0
	defaultRedisAddr    = "localhost:3679"
	defaultTriggerCount = 5
	defaultTimeout      = 30
)

func mustGetEnvVarString(key string) string {
	value, found := os.LookupEnv(key)
	if !found {
		log.Fatalf("[ERROR] required environment variable not found %s", key)
	}
	return value
}

func mustGetEnvVarInt(key string) int {
	stringValue := mustGetEnvVarString(key)
	i, err := strconv.Atoi(stringValue)
	if err != nil {
		log.Fatalf("[ERROR] unable to parse env var %s, expected int but got %s", key, stringValue)
	}
	return i
}

func getEnvVarIntWithDefault(key string, fallback int) int {
	stringValue, found := os.LookupEnv(key)
	if found {
		i, err := strconv.Atoi(stringValue)
		if err != nil {
			log.Fatalf("[ERROR] unable to parse env var %s, expected int but got %s", key, stringValue)
		}
		return i
	} else {
		log.Printf("[DEBUG] var %s not set, using default %d", key, fallback)
		return fallback
	}
}

func getEnvVarStringWithDefault(key string, fallback string) string {
	value, found := os.LookupEnv(key)
	if found {
		return value
	} else {
		log.Printf("[DEBUG] env var %s not set, using default %s", key, fallback)
		return fallback
	}
}

func main() {

	logLevel := getEnvVarStringWithDefault("LOG_LEVEL", "INFO")

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"TRACE", "DEBUG", "WARN", "INFO", "ERROR"},
		MinLevel: logutils.LogLevel(strings.ToUpper(logLevel)),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)

	log.Printf("[DEBUG] log level=%s", filter.MinLevel)

	notificationTimeoutThreshold := time.Duration(getEnvVarIntWithDefault("FOMO_NOTIFICATION_COUNT_TIMEOUT", defaultTimeout)) * time.Second

	repo, err := NewRedisRepository(mustGetEnvVarString("REDIS_ADDR"), mustGetEnvVarString("REDIS_PASSWORD"), getEnvVarIntWithDefault("REDIS_DB", 0), notificationTimeoutThreshold)
	if err != nil {
		log.Fatalf("[ERROR] %s", err)
	}

	slackClient := slack.New(mustGetEnvVarString("SLACK_TOKEN"))

	slackHandler := NewRealSlackHandler(
		repo,
		slackClient,
		mustGetEnvVarString("SLACK_NOTIFICATION_CHANNEL"),
		getEnvVarIntWithDefault("FOMO_NOTIFICATION_COUNT_TRIGGER", defaultTriggerCount),
	)

	log.Print("[DEBUG] checking if the env var LAMBDA_TASK_ROOT exists, if it does, we are in Lambda mode")
	// AWS set the env var LAMBDA_TASK_ROOT, this can be used to test if we are running in AWS or on a server.
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		log.Print("[DEBUG] we are running in Lambda mode")
		lambda.Start(LambdaHandler(slackHandler))
	} else {
		log.Print("[DEBUG] we are running in server mode")
		mux := http.DefaultServeMux
		mux.HandleFunc("/", ServerHandlerFunc(slackHandler))
		panic(http.ListenAndServe(defaultListenAddr, mux))
	}
}

// LambdaHandler returns a function that can be used to recieve lambda events
func LambdaHandler(sh SlackHandler) func(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return func(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
		resp := events.LambdaFunctionURLResponse{}

		var body []byte

		if req.IsBase64Encoded {
			var err error
			body, err = base64.StdEncoding.DecodeString(req.Body)
			if err != nil {
				log.Println(err)
				return resp, err
			}
		} else {
			body = []byte(req.Body)
		}

		handlerResp, err := sh.HandleEvent(body)
		if err != nil {
			log.Printf("[ERROR] %s", err)
			return resp, err
		}

		resp.Body = string(handlerResp.Body)
		resp.StatusCode = handlerResp.StatusCode
		resp.Headers = handlerResp.Headers

		return resp, nil
	}
}

// Enable running the service in standalone server mode
func ServerHandlerFunc(sh SlackHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		resp, err := sh.HandleEvent(body)
		if err != nil {
			log.Printf("[ERROR] failed to handle event: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(resp.StatusCode)
		w.Write(resp.Body)
	}
}
