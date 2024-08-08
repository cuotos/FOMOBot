package main

import (
	"log"
	"net/http"
	"os"

	"github.com/cuotos/fomobot/handler"
)

func init() {
	expectedToken := os.Getenv("AUTH_TOKEN")
	if expectedToken == "" {
		log.Printf("[ERROR] required env var AUTH_TOKEN is not set or empty, this is unsafe")
	}
}

func LeaveChannelHandler(c handler.SlackChannelLeaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		authToken := r.Header.Get("Authentication")

		expectedToken := os.Getenv("AUTH_TOKEN")

		if authToken != expectedToken {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		channel := r.URL.Query().Get("channel")
		if channel == "" {
			log.Println("[WARN] fomobot asked to leave channel, but no channel param provided")
			return
		}

		log.Printf("[TRACE] requested FOMOBot to leave channel %s", channel)
		_, err := c.LeaveConversation(channel)
		if err != nil {
			log.Printf("[ERROR] failed to leave channel: %s", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	}
}
