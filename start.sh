#!/bin/sh

echo "Running database migrations..."
go run main.go migrate

echo "Starting application..."
CompileDaemon -log-prefix=false -build="go build -o main ./cmd/api" -command="./main" 