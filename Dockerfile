# ---------- Build Stage ----------
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .

# 编译 gql 二进制，入口是 gql/graphql/server.go
RUN go build -o gql ./gql/graphql/server.go

# ---------- Run Stage ----------
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/gql .

# 容器启动命令
CMD ["./gql"]
