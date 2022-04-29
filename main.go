package main

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const (
	lambdaMode = true
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	slackHandler := &RealSlackHandler{}

	if lambdaMode {
		lambda.Start(LambdaHandler(slackHandler))
	} else {
		mux := http.DefaultServeMux
		mux.HandleFunc("/", ServerHandlerFunc(slackHandler))
		panic(http.ListenAndServe(":8080", mux))
	}
}

// LambdaHandler returns a function that can be used to recieve lambda events
func LambdaHandler(sh SlackHandler) func(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	return func(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
		resp := events.LambdaFunctionURLResponse{}

		var body []byte

		if req.IsBase64Encoded {
			var err error
			body, err = base64.StdEncoding.DecodeString(req.Body)
			if err != nil {
				log.Println(err)
				return resp, err
			}
		} else {
			body = []byte(req.Body)
		}

		handlerResp, err := sh.HandleEvent(body)
		if err != nil {
			return resp, err
		}

		resp.Body = string(handlerResp.Body)
		resp.StatusCode = handlerResp.StatusCode
		resp.Headers = handlerResp.Headers

		return resp, nil
	}
}

// Enable running the service in standalone api mode
func ServerHandlerFunc(sh SlackHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		resp, err := sh.HandleEvent(body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(resp.StatusCode)
		w.Write(resp.Body)
	}
}
