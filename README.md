# FOMOBot

"never fear of missing out on Slack freebies again!"

FOMOBot is a Slack bot that will let you know if there is something going on that you should know about.

If enough users "emoji response" to a message in a channel within a certain time period, FOMOBot will drop a message in a notification channel to let you know.
This is the only channel you need to pay attention to (via pop up alerts etc)

### why?

There are too many public channels and I do not like watching them all, and when everyone is off to a party or eating free cake I get annoyed at the "didn't you see the message in Slack?"

## How

This app is written to run as a Lambda function or as a long running service


- use ngrok and put url in slack admin page
- uses user permissions to read reactions from all public channels, need to test if this includes channels user is not in
- needs write access as bot, but the bot will need to be in the channel that it wants to write to

- test verification challenge
`GOOS=linux go build -o bin/handler && docker run --rm -ti -v $(pwd):/var/task:ro,delegated -e DOCKER_LAMBDA_DEBUG=true lambci/lambda:go1.x bin/handler "$(cat example_funcurl_challenge.json)"`

- test reaction added
`GOOS=linux go build -o bin/handler && docker run --rm -ti -v $(pwd):/var/task:ro,delegated -e DOCKER_LAMBDA_DEBUG=true lambci/lambda:go1.x bin/handler "$(cat example_funcurl_event.json)"`

- deploy
`GOOS=linux go build -o main && zip function.zip main &&  av exec personal -- aws lambda  update-function-code --function-name fomobot --zip-file fileb://function.zip`