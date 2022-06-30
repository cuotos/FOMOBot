package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type SlackClient interface {
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
	Repository          Repository
	SlackClient         SlackClient
}

func NewRealSlackHandler(repo Repository, slackClient SlackClient, notificationChannel string, reacitonThreshold int) SlackHandler {
	return &RealSlackHandler{
		NotificationChannel: notificationChannel,
		ReactionThreshold:   reacitonThreshold,
		Repository:          repo,
		SlackClient:         slackClient,
	}
}

func (sh *RealSlackHandler) HandleEvent(body []byte) (SlackHandlerResponse, error) {
	resp := SlackHandlerResponse{}

	event, err := slackevents.ParseEvent(body, slackevents.OptionNoVerifyToken()) //TODO: verify
	if err != nil {
		resp.StatusCode = http.StatusInternalServerError
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
			val, err := sh.Repository.Incr(context.Background(), fmt.Sprintf("%s_%s_%s", event.TeamID, e.Item.Channel, e.Item.Timestamp))
			if err != nil {
				resp.StatusCode = http.StatusInternalServerError
				return resp, err
			}

			if val == sh.ReactionThreshold {
				messageLink, err := sh.SlackClient.GetPermalink(&slack.PermalinkParameters{
					Channel: e.Item.Channel,
					Ts:      e.Item.Timestamp,
				})
				if err != nil {
					return resp, fmt.Errorf("unable to get permalink for event: %w", err)
				}

				msgText := fmt.Sprintf("This message appears to be getting plenty of attention: %s", messageLink)

				err = sh.SendMessage(sh.NotificationChannel, msgText) // TODO: put together message
				if err != nil {
					return resp, fmt.Errorf("failed to send message to slack: %w", err)
				}
			}
		}
	}

	return resp, nil
}

func (sh *RealSlackHandler) SendMessage(channel string, msg string) error {
	_, _, _, err := sh.SlackClient.SendMessage(channel, slack.MsgOptionText(msg, false))
	return err
}
