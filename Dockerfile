# ---------- Step 1: Build ----------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server ./gql/graphql/server.go

# ---------- Step 2: Runtime ----------
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
