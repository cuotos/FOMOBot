- use ngrok and put url in slack admin page
- uses user permissions to read reactions from all public channels, need to test if this includes channels user is not in
- needs write access as bot, but the bot will need to be in the channel that it wants to write to

- test verification challenge
`GOOS=linux go build -o bin/handler && docker run --rm -ti -v $(pwd):/var/task:ro,delegated -e DOCKER_LAMBDA_DEBUG=true lambci/lambda:go1.x bin/handler "$(cat example_funcurl_challenge.json)"`

- test reaction added
`GOOS=linux go build -o bin/handler && docker run --rm -ti -v $(pwd):/var/task:ro,delegated -e DOCKER_LAMBDA_DEBUG=true lambci/lambda:go1.x bin/handler "$(cat example_funcurl_event.json)"`

- deploy
`GOOS=linux go build -o main && zip function.zip main &&  av exec personal -- aws lambda  update-function-code --function-name fomobot --zip-file fileb://function.zip`