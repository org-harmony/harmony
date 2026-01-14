# First stage: build the application
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod, go.sum and vendor directory
COPY go.mod go.sum ./
COPY vendor/ vendor/

# Copy the source code
COPY src/ src/

# Build the 'web' and 'migrate' executables
RUN CGO_ENABLED=0 GOOS=linux go build -o web src/cmd/web/*.go
RUN CGO_ENABLED=0 GOOS=linux go build -o migrate src/cmd/migrate/*.go

# Second stage: setup the runtime environment
FROM alpine:latest

WORKDIR /app

# Copy the pre-built binary files from the previous stage
COPY --from=builder /app/web .
COPY --from=builder /app/migrate .

# Copy configuration files and directories
COPY config/ config/
COPY migrations/ migrations/
COPY public/ public/
COPY templates/ templates/
COPY translations/ translations/
COPY docs/templates/paris/ docs/templates/paris/

# Set environment variables
ENV PORT=8213

# Expose the application on port 8213
EXPOSE 8213

# Command to run the 'web' executable
ENTRYPOINT ["./web"]
