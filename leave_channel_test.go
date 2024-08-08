package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockSlackChannelLeaver struct {
	leaveFn func(string)
}

func (mscl MockSlackChannelLeaver) LeaveConversation(channel string) (bool, error) {
	mscl.leaveFn(channel)
	return false, nil
}

func TestLeaveChannelHandler(t *testing.T) {
	var providedChannel string

	channelLeavaer := &MockSlackChannelLeaver{
		leaveFn: func(s string) {
			providedChannel = s
		},
	}

	h := LeaveChannelHandler(channelLeavaer)

	inputChannelName := "something"

	mockRequest, _ := http.NewRequest("GET", "https://localhostthing?channel="+inputChannelName, nil)
	h(nil, mockRequest)

	assert.Equal(t, providedChannel, inputChannelName)
}
