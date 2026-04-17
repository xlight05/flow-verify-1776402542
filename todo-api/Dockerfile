FROM golang:1.22 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/todo-api .

FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /out/todo-api /app/todo-api
EXPOSE 9090
VOLUME ["/data"]
USER nonroot:nonroot
ENTRYPOINT ["/app/todo-api"]
