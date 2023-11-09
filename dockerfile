FROM golang:1.20 as golang

WORKDIR /app

COPY ./ ./

RUN go mod tidy

CMD ["/bin/bash","-c","go run . --configPath=config.yaml"]