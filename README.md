# Market Control System

Backend service for the Market Control System platform.

## Features

- Project Management (support for multiple parallel projects)
- Address Lifecycle Management (batch generation, tagging, status monitoring)
- Gas Distribution Mechanism (unified management, periodic automatic transfers)
- Strategy Management and Scheduling (different strategies for each address set)
- Real-time Token Balance Monitoring for Multiple Addresses

## Tech Stack

- Go 1.21+
- Gin Web Framework
- GORM
- PostgreSQL
- Docker & Docker Compose

## Requirements

- Docker
- Docker Compose

## Running with Docker

1. Clone the repository
   
   ```bash
   git clone [repository-url]
   cd marketcontrol
   ```

2. Build and run containers
   
   ```bash
   docker-compose up --build
   ```

3. Run containers only (if already built)
   
   ```bash
   docker-compose up
   ```

4. Run containers in background
   
   ```bash
   docker-compose up -d
   ```

5. Stop containers
   
   ```bash
   docker-compose down
   ```

## Environment Variables

Docker Compose is configured with the following environment variables:

```env
DB_HOST=postgres
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=marketcontrol
DB_PORT=5432
PORT=8080
```

## API Documentation

### Blockchain Config API

- `POST /blockchain-config` - Create a new blockchain configuration
- `GET /blockchain-config/:id` - Get a specific blockchain configuration
- `GET /blockchain-config` - Get all blockchain configurations
- `PUT /blockchain-config/:id` - Update a blockchain configuration
- `DELETE /blockchain-config/:id` - Delete a blockchain configuration

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── models/
│   ├── repository/
│   ├── service/
│   ├── handler/
│   └── middleware/
├── pkg/
│   ├── config/
│   └── utils/
├── configs/
├── Dockerfile
└── docker-compose.yml
```

```
go run scripts/run_wash_task.go -manage-id=
go run scripts/export_address_nodes.go -map-id 33 -node-type leaf -file-name test_export
```