package main

import (
	"fmt"

	"github.com/slack-go/slack"
)

type SlackSender struct {
	client *slack.Client
}

func NewSlackSender(token string) (*SlackSender, error) {
	client := slack.New(token, slack.OptionDebug(false))
	_, err := client.AuthTest()

	if err != nil {
		return nil, fmt.Errorf("auth failed: %w", err)
	}

	return &SlackSender{
		client: client,
	}, nil
}

func (s *SlackSender) SendMessage(channel string, message string) error {
	_, _, err := s.client.PostMessage(channel, slack.MsgOptionText(message, false))
	return err
}
