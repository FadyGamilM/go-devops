FROM golang:1.21.7-alpine AS builder

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest  

WORKDIR /root/

COPY --from=builder /app/main .

ENV HOST ""
ENV SERVER_PATH ""
ENV MONGO_URI ""

CMD ["./main"]