FROM golang:1.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -mod=readonly -o /fomobot ./...


FROM public.ecr.aws/lambda/provided:al2023

COPY --from=builder /fomobot ./main

ENTRYPOINT [ "./main" ]