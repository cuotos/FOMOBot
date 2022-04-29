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

	event, err := slackevents.ParseEvent(body, slackevents.OptionNoVerifyToken())
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
			val, err := sh.Repository.Incr(context.Background(), fmt.Sprintf("%s_%s", e.Item.Channel, e.Item.Timestamp))
			if err != nil {
				resp.StatusCode = http.StatusInternalServerError
				return resp, err
			}

			if val == sh.ReactionThreshold {
				sh.SendMessage(sh.NotificationChannel, "") // TODO: put together message
			}
		}
	}

	return resp, nil
}

func (sh *RealSlackHandler) SendMessage(channel string, _ string) error {
	_, _, _, err := sh.SlackClient.SendMessage(channel)
	return err
}
