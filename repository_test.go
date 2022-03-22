package main

import (
	"testing"

	"github.com/slack-go/slack/slackevents"
	"github.com/stretchr/testify/assert"
)

func TestGenerateKey(t *testing.T) {
	tcs := []struct {
		InputEvent  *slackevents.ReactionAddedEvent
		ExpectedKey string
	}{
		{
			&slackevents.ReactionAddedEvent{
				Item: slackevents.Item{
					Channel:   "C02NG8RM10R",
					Timestamp: "1647975399.425609",
				},
			},
			"C02NG8RM10R_1647975399.425609",
		},
	}

	for _, tc := range tcs {
		actual := GenerateKey(tc.InputEvent)
		assert.Equal(t, tc.ExpectedKey, actual)
	}
}
