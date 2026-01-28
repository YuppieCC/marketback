# use the official golang image
FROM golang:1.21-alpine

# set the working directory
WORKDIR /app

# install necessary dependencies
RUN apk add --no-cache gcc musl-dev git wget

# expose port
EXPOSE 8080

# Use CompileDaemon to enable hot reload
RUN go install github.com/githubnemo/CompileDaemon@latest

# Create startup script
COPY start.sh /app/start.sh
RUN chmod +x /app/start.sh && \
    ls -l /app/start.sh

# start the project
CMD ["/bin/sh", "/app/start.sh"]