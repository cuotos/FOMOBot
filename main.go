package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"

	"github.com/hashicorp/logutils"
	"github.com/joho/godotenv"
)

const (
	//notificationThresholdPeriod time.Duration = time.Hour
	notificationThresholdPeriod time.Duration = time.Second * 30
	notificationThresholdCount  int           = 2
)

var (
	SLACK_TOKEN                   string
	SLACK_SIGNING_SECRET          string
	SLACK_NOTIFICATION_CHANNEL_ID string
	NOTIFICATION_THRESHOLD        int
	VERIFICATION_TOKEN            string
)

var (
	redisRepo   Repository
	slackClient *slack.Client
)

func mustGetenv(variable string) string {
	found := os.Getenv(variable)
	if found == "" {
		log.Fatalf("mising required env var %s", variable)
	}
	return found
}

func main() {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
		MinLevel: logutils.LogLevel("DEBUG"),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := run(); err != nil {
		log.Fatal("[FATAL] ", err)
	}
}

func run() error {

	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("failed to load .env file: %w\n", err)
	}

	// TODO: replace with a proper env parsing framework like "github.com/kelseyhightower/envconfig"
	SLACK_TOKEN = mustGetenv("SLACK_TOKEN")
	SLACK_SIGNING_SECRET = mustGetenv("SLACK_SIGNING_SECRET")
	SLACK_NOTIFICATION_CHANNEL_ID = mustGetenv("SLACK_NOTIFICATION_CHANNEL_ID")
	VERIFICATION_TOKEN = mustGetenv("VERIFICATION_TOKEN")

	thresholdString := mustGetenv("NOTIFICATION_THRESHOLD")
	thresholdInt, err := strconv.Atoi(thresholdString)
	NOTIFICATION_THRESHOLD = thresholdInt
	if err != nil {
		return fmt.Errorf("NOTIFICATION_THRESHOLD must be an integer. but got \"%s\"\n", thresholdString)
	}

	slackClient = slack.New(SLACK_TOKEN)
	_, err = slackClient.AuthTest()
	if err != nil {
		return fmt.Errorf("failed to create slack client: %w", err)
	}

	//TODO: get redis config from env
	redisRepo, err = NewRedisRepository("localhost:6379", "", 0, notificationThresholdPeriod)
	if err != nil {
		return fmt.Errorf("failed to create redis client: %w", err)
	}

	http.HandleFunc("/healthz", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("ok"))
	})

	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("[ERROR] ", err)
			return
		}

		sv, err := slack.NewSecretsVerifier(r.Header, SLACK_SIGNING_SECRET)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("[ERROR] ", err)
			return
		}

		if _, err := sv.Write(body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("[ERROR] ", err)
			return
		}

		if err := sv.Ensure(); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Println("[ERROR] ", err)
			return
		}

		// Token verifier compares the token supplied with the message with the one Slack give you in the UI off the app.
		slackMessageVerifier := slackevents.TokenComparator{VerificationToken: VERIFICATION_TOKEN}
		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(slackMessageVerifier))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("[ERROR] ", err)
			return
		}

		// Verification process carried out by Slack when you add destination to app webhook config.
		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println("[ERROR] ", err)
				return
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {

			switch event := eventsAPIEvent.InnerEvent.Data.(type) {
			// TODO: dispatch handling a reaction even to a go routine in order to finish the web call quick.
			// needs to be a single routine in order to serialize the Incr calls to redis. although, they are atomic.... :thinking:
			case *slackevents.ReactionAddedEvent:

				redisKey := fmt.Sprintf("%s_%s", event.Item.Channel, event.Item.Timestamp)

				val, err := redisRepo.Incr(context.Background(), redisKey)
				if err != nil {
					log.Printf("[ERROR] failed to write to Redis: %s\n", err)
				}

				if val == thresholdInt {
					messageLink, err := slackClient.GetPermalink(&slack.PermalinkParameters{
						Channel: event.Item.Channel,
						Ts:      event.Item.Timestamp,
					})
					if err != nil {
						log.Printf("unable to get permalink for interesting message: %s\n", err)
					}

					slackMsgOpts := []slack.MsgOption{}
					slackMsgOpts = append(slackMsgOpts, slack.MsgOptionText(fmt.Sprintf("This message is getting plenty of attention: %s", messageLink), false))

					_, _, _, err = slackClient.SendMessage(SLACK_NOTIFICATION_CHANNEL_ID, slackMsgOpts...)
					if err != nil {
						log.Printf("[ERROR] %s", err)
					}
				}

			case *slackevents.ReactionRemovedEvent:
				log.Println("[DEBUG] reaction removed: ", event.Reaction)
			default:
				log.Printf("[DEBUG] unknown event %T", event)
			}
		}

		return
	})

	log.Println("[INFO] Server listening")

	return http.ListenAndServe(":3000", nil)
}
