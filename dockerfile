FROM golang:1.23.0-alpine

RUN apk update && apk upgrade && \
    apk add --no-cache bash openssh

WORKDIR /app

RUN mkdir -p /app/monitoring-system/monolith
WORKDIR /app/monitoring-system/monolith

COPY . .

RUN go mod download && go mod verify

RUN go build -o /bin/monolith /app/monitoring-system/monolith/cmd/main.go

EXPOSE 13693

CMD ["/bin/monolith"]