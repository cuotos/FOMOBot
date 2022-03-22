package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"

	"github.com/hashicorp/logutils"
	"github.com/joho/godotenv"
)

var (
	SLACK_TOKEN          string
	SLACK_SIGNING_SECRET string
)

var redisRepo *RedisRepository

func mustGetenv(variable string) string {
	found := os.Getenv(variable)
	if found == "" {
		log.Fatalf("mising required env var %s", variable)
	}
	return found
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	SLACK_TOKEN = mustGetenv("SLACK_TOKEN")
	SLACK_SIGNING_SECRET = mustGetenv("SLACK_SIGNING_SECRET")

	//client := slack.New(SLACK_TOKEN, slack.OptionDebug(false))
	redisRepo, err = NewRedisRepository()
	if err != nil {
		log.Fatalf("[ERROR] failed to create redis client: %s", err)
	}

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("DEBUG"),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	http.HandleFunc("/healthz", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("ok"))
	})

	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("[ERROR]", err)
			return
		}

		// fmt.Printf("[DEBUG] %s\n", body)

		sv, err := slack.NewSecretsVerifier(r.Header, SLACK_SIGNING_SECRET)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("[ERROR]", err)
			return
		}

		if _, err := sv.Write(body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("[ERROR]", err)
			return
		}

		if err := sv.Ensure(); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Println("[ERROR]", err)
			return
		}

		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("[ERROR]", err)
			return
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println("[ERROR]", err)
				return
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))
		}

		if eventsAPIEvent.Type == slackevents.CallbackEvent {

			switch event := eventsAPIEvent.InnerEvent.Data.(type) {
			case *slackevents.ReactionAddedEvent:
				err := redisRepo.Incr(GenerateKey(event))
				if err != nil {
					log.Printf("[ERROR] failed to write to Redis: %s\n", err)
				}

				val, err := redisRepo.Get(GenerateKey(event))
				if err != nil {
					log.Printf("[ERROR] failed to read from redis: %s\n", err)
				}

				log.Printf("[INFO] %v\n", val)
				// log.Println("[DEBUG] detected reaction, replying")
				// _, _, err := client.PostMessage(event.Item.Channel, slack.MsgOptionTS(event.Item.Timestamp), slack.MsgOptionText("This is a reply", false))
				// if err != nil {
				// 	log.Println("[ERROR]", err)
				// }

			case *slackevents.ReactionRemovedEvent:
				log.Println("[DEBUG]", event.Reaction)
			default:
				log.Printf("[DEBUG] unknown event %T", event)
			}
		}
	})

	log.Println("[INFO] Server listening")

	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}
}
