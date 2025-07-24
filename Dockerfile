FROM golang:1.24 AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/

# Create directory structure that will be copied to final image
RUN mkdir -p /build/config
RUN mkdir -p /build/can

# ./cmd is the folder where the main.go file is located
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd

FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /build/app .

# Copy the directory structure from the builder stage
COPY --from=builder /build/config ./config
COPY --from=builder /build/can ./can

# Expose default UDP port
EXPOSE 20000/udp

# Run the application
CMD ["./app"]