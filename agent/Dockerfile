FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY . .
RUN go build -o /out/ops-agent ./cmd/agent

FROM alpine:3.22

RUN addgroup -S ops && adduser -S ops -G ops
USER ops
WORKDIR /app

COPY --from=builder /out/ops-agent /app/ops-agent

EXPOSE 18080
CMD ["/app/ops-agent"]
