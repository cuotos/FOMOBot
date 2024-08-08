package main

import (
	"context"
	"encoding/base64"
	"io"
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

	"github.com/cuotos/fomobot/database"
	"github.com/cuotos/fomobot/handler"
)

const (
	defaultListenAddr   = "0.0.0.0:8080"
	defaultTriggerCount = 5  // number of reactions
	defaultTimeout      = 30 // in this many seconds
)

func main() {

	logLevel := getenvStrWithDefault("LOG_LEVEL", "INFO")

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"TRACE", "DEBUG", "WARN", "INFO", "ERROR"},
		MinLevel: logutils.LogLevel(strings.ToUpper(logLevel)),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)

	log.Printf("[DEBUG] log level=%s", filter.MinLevel)

	notificationTimeoutThreshold := time.Duration(getenvIntWithDefault("FOMO_NOTIFICATION_COUNT_TIMEOUT", defaultTimeout)) * time.Second

	db, err := database.NewRedisDatabase(mustGetenvStr("REDIS_ADDR"), getenvStrWithDefault("REDIS_PASSWORD", ""), getenvIntWithDefault("REDIS_DB", 0), notificationTimeoutThreshold)
	if err != nil {
		log.Fatalf("[ERROR] %s", err)
	}

	slackClient := slack.New(mustGetenvStr("SLACK_TOKEN"))

	slackHandler := handler.NewRealSlackHandler(
		db,
		slackClient,
		mustGetenvStr("SLACK_NOTIFICATION_CHANNEL"),
		getenvIntWithDefault("FOMO_NOTIFICATION_COUNT_TRIGGER", defaultTriggerCount),
	)

	log.Print("[DEBUG] checking if the env var AWS_LAMBDA_RUNTIME_API exists, if it does, we are in Lambda mode")

	// AWS set the env var AWS_LAMBDA_RUNTIME_API, this can be used to test if we are running in AWS or on a server.
	if weRunningInALambda() {

		log.Print("[DEBUG] running in Lambda mode")
		lambda.Start(LambdaHandler(slackHandler))

	} else {

		log.Print("[DEBUG] running in server mode")
		mux := http.DefaultServeMux
		mux.HandleFunc("/", ServerHandlerFunc(slackHandler))
		mux.HandleFunc("/healthz", HealthCheckHandler())
		mux.HandleFunc("/leave", LeaveChannelHandler(slackClient))

		listenAdd := getenvStrWithDefault("LISTEN", defaultListenAddr)
		log.Printf("[INFO] server running on %s", listenAdd)
		panic(http.ListenAndServe(listenAdd, mux))
	}
}

// LambdaHandler returns a function that can be used to recieve lambda events
func LambdaHandler(sh handler.SlackHandler) func(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
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
func ServerHandlerFunc(sh handler.SlackHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp, err := sh.HandleEvent(body)
		if err != nil {
			log.Printf("[ERROR] failed to handle event: %s", err)
			w.WriteHeader(resp.StatusCode)
			w.Write(resp.Body)
			return
		}

		w.WriteHeader(resp.StatusCode)
		w.Write(resp.Body)
	}
}

func mustGetenvStr(key string) string {
	value, found := os.LookupEnv(key)
	if !found {
		log.Fatalf("[ERROR] required environment variable not found %s", key)
	}
	return value
}

func getenvIntWithDefault(key string, fallback int) int {
	if stringValue := os.Getenv(key); stringValue != "" {
		i, err := strconv.Atoi(stringValue)
		if err != nil {
			log.Fatalf("[ERROR] unable to parse env var %s, expected int but got %s", key, stringValue)
		}
		return i
	}

	log.Printf("[DEBUG] var %s not set, using default %d", key, fallback)
	return fallback

}

func getenvStrWithDefault(key string, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	log.Printf("[DEBUG] env var %s not set, using default %s", key, fallback)
	return fallback

}

func weRunningInALambda() bool {
	return os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""
}
