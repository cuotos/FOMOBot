package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/cuotos/fomobot/database"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type SlackChannelLeaver interface {
	LeaveConversation(string) (bool, error)
}

type SlackClient interface {
	SlackChannelLeaver
	SendMessage(string, ...slack.MsgOption) (string, string, string, error)
	GetPermalink(*slack.PermalinkParameters) (string, error)
}

type SlackHandlerResponse struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
}

type SlackHandler interface {
	HandleEvent([]byte) (SlackHandlerResponse, error)
	SendMessage(string, string) error
}

type RealSlackHandler struct {
	NotificationChannel string
	ReactionThreshold   int
	Database            database.Database
	SlackClient         SlackClient
}

func NewRealSlackHandler(db database.Database, slackClient SlackClient, notificationChannel string, reacitonThreshold int) SlackHandler {
	return &RealSlackHandler{
		NotificationChannel: notificationChannel,
		ReactionThreshold:   reacitonThreshold,
		Database:            db,
		SlackClient:         slackClient,
	}
}

func (sh *RealSlackHandler) HandleEvent(body []byte) (SlackHandlerResponse, error) {
	log.Printf("[TRACE] handling event: %s", body)

	// create an empty default response
	resp := SlackHandlerResponse{
		StatusCode: http.StatusBadRequest,
		Body:       []byte{},
		Headers:    map[string]string{},
	}

	if len(body) == 0 {
		log.Println("[DEBUG] no body provided in request")
		return resp, nil
	}

	event, err := slackevents.ParseEvent(body, slackevents.OptionNoVerifyToken()) //TODO: verify
	if err != nil {
		resp.StatusCode = http.StatusBadRequest
		resp.Body = []byte(err.Error())
		return resp, err
	}

	switch event.Type {

	case slackevents.URLVerification:
		challengeResponse := &slackevents.ChallengeResponse{}
		err := json.Unmarshal(body, challengeResponse)
		if err != nil {
			resp.StatusCode = http.StatusInternalServerError
			return resp, err
		}

		resp.Headers = map[string]string{"Content-Type": "text"}
		resp.Body = []byte(challengeResponse.Challenge)
		resp.StatusCode = http.StatusOK

	case slackevents.CallbackEvent:
		switch e := event.InnerEvent.Data.(type) {
		case *slackevents.ReactionAddedEvent:
			resp.StatusCode = http.StatusOK
			val, err := sh.Database.Incr(context.Background(), fmt.Sprintf("%s_%s_%s", event.TeamID, e.Item.Channel, e.Item.Timestamp))
			if err != nil {
				resp.StatusCode = http.StatusInternalServerError
				return resp, err
			}

			if val == sh.ReactionThreshold {
				log.Printf("[DEBUG] sending slack notification message to channel %s", sh.NotificationChannel)
				messageLink, err := sh.SlackClient.GetPermalink(&slack.PermalinkParameters{
					Channel: e.Item.Channel,
					Ts:      e.Item.Timestamp,
				})
				if err != nil {
					return resp, fmt.Errorf("unable to get permalink for event: %w", err)
				}
				log.Printf("[TRACE] sending slack notification message regarding message %s %s", e.Item.Channel, e.Item.Timestamp)

				msgText := fmt.Sprintf("This message appears to be getting plenty of attention: <%s|here>", messageLink)

				err = sh.SendMessage(sh.NotificationChannel, msgText) // TODO: put together message
				if err != nil {
					return resp, fmt.Errorf("failed to send message to slack: %w", err)
				}
			}
		}
	default:
		resp.StatusCode = http.StatusBadRequest
		resp.Body = []byte("unknown event type")
		return resp, errors.New("invalid request. unknown Slack event type")
	}

	return resp, nil
}

func (sh *RealSlackHandler) SendMessage(channel string, msg string) error {
	_, _, _, err := sh.SlackClient.SendMessage(channel, slack.MsgOptionText(msg, false))
	return err
}
