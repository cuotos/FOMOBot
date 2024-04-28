package main

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/cuotos/fomobot/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Testing happy path only...
func TestCorrectlyDecodesBase64Requests(t *testing.T) {
	tcs := []struct {
		InputBody string
		IsBase64  bool
		Expected  string
	}{
		{
			"dGhpc19pc19hX3Rlc3RfYm9keQ==",
			true,
			"this_is_a_test_body",
		},
		{
			"plain_text_string",
			false,
			"plain_text_string",
		},
	}

	for _, tc := range tcs {
		mockSlackHandler := MockSlackHandler{}

		lambdaFunc := LambdaHandler(mockSlackHandler)

		mockRequest := events.LambdaFunctionURLRequest{
			Body:            tc.InputBody,
			IsBase64Encoded: tc.IsBase64,
		}

		actualResp, err := lambdaFunc(context.Background(), mockRequest)

		require.NoError(t, err)
		assert.Equal(t, tc.Expected, actualResp.Body)
	}
}

func TestBadBase64ReturnsAnError(t *testing.T) {
	mockSlackHandler := MockSlackHandler{}
	lambdaFunc := LambdaHandler(mockSlackHandler)
	mockRequest := events.LambdaFunctionURLRequest{
		Body:            "this_is_bad_base64",
		IsBase64Encoded: true,
	}
	_, err := lambdaFunc(context.Background(), mockRequest)

	assert.Error(t, err)
}

type MockSlackHandler struct{}

func (msh MockSlackHandler) HandleEvent(body []byte) (handler.SlackHandlerResponse, error) {
	// just echo back the body that was passed in to check if it was decoded from base64 before calling Handle
	return handler.SlackHandlerResponse{Body: body}, nil
}

func (msh MockSlackHandler) SendMessage(_, _ string) error {
	return nil
}
