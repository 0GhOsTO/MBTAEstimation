FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o mbta-backend .

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /app/mbta-backend /app/mbta-backend
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/mbta-backend"]
