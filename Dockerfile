FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o trace2prompt .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/trace2prompt .
EXPOSE 4318 4319
CMD ["./trace2prompt"]