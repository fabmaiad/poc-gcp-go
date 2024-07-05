FROM golang:1.22.3

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

#RUN go build -o main .

ENV PROJECT_ID=bullla-one-d-apps-cn-92c3
ENV TOPIC_ID=topic1-poc-golang
ENV ENDPOINT_SERVER=http://localhost:9090/v1/json-server/gets
ENV LOCAL_ONLY=false
ENV FUNCTION_TARGET=Main

EXPOSE 8081

CMD ["sh", "-c", "go run cmd/main.go"]