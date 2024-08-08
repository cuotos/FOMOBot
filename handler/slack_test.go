package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/cuotos/fomobot/database"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLVerificationRequest(t *testing.T) {
	mockChallengeBody := []byte(`{
		"token": "Jhj5dZrVaK7ZwHHjRyZWjbDl",
		"challenge": "3eZbrw1aBm2rZgRNXXXXXXXXX9CY3gmdALWMmHkvFXO7tYXAYM8P",
		"type": "url_verification"
	}`)

	handler := &RealSlackHandler{}
	actualResponse, err := handler.HandleEvent(mockChallengeBody)

	require.NoError(t, err)

	assert.Equal(t, "3eZbrw1aBm2rZgRNXXXXXXXXX9CY3gmdALWMmHkvFXO7tYXAYM8P", string(actualResponse.Body))
	assert.Equal(t, "text", actualResponse.Headers["Content-Type"])
	assert.Equal(t, http.StatusOK, actualResponse.StatusCode)
}

func TestReactionAddedEventCallsTheDBIncr(t *testing.T) {
	var actualKey string
	calledCount := 0
	mr := NewMockDB(
		// incrFunc
		func(key string) (int, error) {
			actualKey = key
			calledCount++
			return calledCount, nil
		},
	)

	msc := &MockSlackClient{}

	handler := NewRealSlackHandler(mr, msc, "testNotificationChannelID", 3)

	actualResponse, err := handler.HandleEvent([]byte(mockReactionAddedEventJSON))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, actualResponse.StatusCode)
	require.False(t, msc.messageSent)

	actualResponse, _ = handler.HandleEvent([]byte(mockReactionAddedEventJSON))

	actualResponse, err = handler.HandleEvent([]byte(mockReactionAddedEventJSON))
	require.NoError(t, err)
	require.True(t, msc.messageSent)
	assert.Equal(t, "T01XXXXKPC_C0XXXXX10R_1648042128.021399", actualKey)
	assert.Equal(t, "testNotificationChannelID", msc.channelCalled)
}

func TestSendsMessageToSlackWhenCorrectNumberOfReactionsOccured(t *testing.T) {
	msc := &MockSlackClient{}
	calledCount := 0
	mr := NewMockDB(
		func(s string) (int, error) {
			calledCount++
			return calledCount, nil
		},
	)

	handler := NewRealSlackHandler(mr, msc, "testNotificationChannelID", 2)

	// call the handler, check if a message was sent, and reset the trigger
	handler.HandleEvent([]byte(mockReactionAddedEventJSON))
	assert.False(t, msc.messageSent)
	msc.messageSent = false

	handler.HandleEvent([]byte(mockReactionAddedEventJSON))
	assert.True(t, msc.messageSent)
	msc.messageSent = false

	handler.HandleEvent([]byte(mockReactionAddedEventJSON))
	assert.False(t, msc.messageSent)
	msc.messageSent = false
}

type MockSlackClient struct {
	messageSent   bool
	channelCalled string
}

func (msc *MockSlackClient) SendMessage(ch string, _ ...slack.MsgOption) (string, string, string, error) {
	msc.messageSent = true
	// set the field on the mock to the channel name that was actually called to verify it was correct
	msc.channelCalled = ch

	return "", "", "", nil
}

func (msc *MockSlackClient) GetPermalink(_ *slack.PermalinkParameters) (string, error) {
	return "", nil
}

func (msc *MockSlackClient) LeaveConversation(_ string) (bool, error) {
	return false, nil
}

var mockReactionAddedEventJSON = `
{
  "token":"jkVTna0zzT7SDuwLHQyEIsjy",
  "team_id":"T01XXXXKPC",
  "api_app_id":"A02NTAM71KJ",
  "event":{
    "type":"reaction_added",
    "user":"U01FXXXXXE",
    "reaction":"white_check_mark",
    "item":{
      "type":"message",
      "channel":"C0XXXXX10R",
      "ts":"1648042128.021399"
    },
    "item_user":"U01F0TT45QE",
    "event_ts":"1650644028.002900"
  },
  "type":"event_callback",
  "event_id":"Ev03CDBHC162",
  "event_time":1650644028,
  "authorizations":[
    {
      "enterprise_id":null,
      "team_id":"T01XXXXKPC",
      "user_id":"U01FXXXXXE",
      "is_bot":true,
      "is_enterprise_install":false
    }
  ],
  "is_ext_shared_channel":false,
  "event_context":"4-eyJldCI6InJlYWN0aW9uX2FkZGVkIiwidGlkXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXDJOVEFNNzFLSiIsImNpZCI6IkMwMk5HOFJNMTBSIn0"
}
`

func NewMockDB(fn func(string) (int, error)) database.Database {
	return &MockDB{
		incrFunc: fn,
	}
}

type MockDB struct {
	incrFunc func(string) (int, error)
}

func (mr MockDB) Incr(_ context.Context, key string) (int, error) {
	return mr.incrFunc(key)
}

func (mr MockDB) Healthy(_ context.Context) error {
	return nil
}
