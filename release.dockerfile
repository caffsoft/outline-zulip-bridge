FROM golang:1.24.1-alpine3.21 AS build-stage
LABEL authors="marc"

# Set our destination for copy
WORKDIR /app

# Download Go modules
COPY go.mod ./
# If/when there are other modules, add go.sum to the above copy line and uncomment the next line
# RUN go mod download

# Copy the source code
COPY . .
# COPY pkg .

# Build the binary and the migrator
RUN env CGO_ENABLED=0 GOOS=linux go build -o /outline-zulip-bridge .

# Deploy application binary into the smallest image possible
FROM alpine:3.21 AS build-release-stage

WORKDIR /

RUN mkdir /app

COPY --from=build-stage /outline-zulip-bridge /app/outline-zulip-bridge

# Expose our ports
EXPOSE 8484

# Run it
RUN adduser -D app
USER app

CMD [ "/app/outline-zulip-bridge" ]
