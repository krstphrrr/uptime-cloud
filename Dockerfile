# Use the official Golang image
FROM golang:1.23.4

# Set the working directory
WORKDIR /app

# Copy files
COPY . .

# Install dependencies and build the binary
RUN go mod tidy && go build -o uptime_monitor

# Expose the Prometheus metrics port
EXPOSE 8080

# Run the application
CMD ["./uptime_monitor"]
