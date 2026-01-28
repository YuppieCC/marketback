#!/bin/bash

# Set color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if golang-migrate is installed
if ! command -v migrate &> /dev/null; then
    echo -e "${RED}Error: golang-migrate is not installed${NC}"
    echo "Please install it using:"
    echo "brew install golang-migrate"
    exit 1
fi

# Set migrations directory
MIGRATIONS_DIR="migrations"

# Create migrations directory if it doesn't exist
mkdir -p $MIGRATIONS_DIR

# Show help information
show_help() {
    echo "Usage: $0 [command]"
    echo
    echo "Commands:"
    echo "  create <name>    Create a new migration"
    echo "  up              Run all pending migrations"
    echo "  down            Rollback the last migration"
    echo "  version         Show current migration version"
    echo "  force <version> Force migration version"
    echo "  help            Show this help message"
}

# Create a new migration
create_migration() {
    if [ -z "$1" ]; then
        echo -e "${RED}Error: Migration name is required${NC}"
        show_help
        exit 1
    fi

    echo -e "${YELLOW}Creating new migration: $1${NC}"
    migrate create -ext sql -dir $MIGRATIONS_DIR -seq "$1"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Migration files created successfully${NC}"
        echo "Please edit the following files:"
        echo "  $MIGRATIONS_DIR/*_${1}.up.sql"
        echo "  $MIGRATIONS_DIR/*_${1}.down.sql"
    else
        echo -e "${RED}Failed to create migration files${NC}"
        exit 1
    fi
}

# Run all pending migrations
run_migrations() {
    echo -e "${YELLOW}Running pending migrations...${NC}"
    migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" up
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Migrations completed successfully${NC}"
    else
        echo -e "${RED}Migration failed${NC}"
        exit 1
    fi
}

# Rollback the last migration
rollback_migration() {
    echo -e "${YELLOW}Rolling back last migration...${NC}"
    migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" down 1
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Rollback completed successfully${NC}"
    else
        echo -e "${RED}Rollback failed${NC}"
        exit 1
    fi
}

# Show current migration version
show_version() {
    echo -e "${YELLOW}Current migration version:${NC}"
    migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" version
}

# Force migration version
force_version() {
    if [ -z "$1" ]; then
        echo -e "${RED}Error: Version number is required${NC}"
        show_help
        exit 1
    fi

    echo -e "${YELLOW}Forcing migration version to $1...${NC}"
    migrate -path $MIGRATIONS_DIR -database "$DATABASE_URL" force $1
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Version forced successfully${NC}"
    else
        echo -e "${RED}Failed to force version${NC}"
        exit 1
    fi
}

# Load environment variables from .env file
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Check environment variables
if [ -z "$DATABASE_URL" ]; then
    echo -e "${RED}Error: DATABASE_URL environment variable is not set${NC}"
    echo "Please set it in your .env file or using:"
    echo "export DATABASE_URL='postgres://username:password@localhost:5432/dbname?sslmode=disable'"
    exit 1
fi

# Main command handler
case "$1" in
    "create")
        create_migration "$2"
        ;;
    "up")
        run_migrations
        ;;
    "down")
        rollback_migration
        ;;
    "version")
        show_version
        ;;
    "force")
        force_version "$2"
        ;;
    "help"|"")
        show_help
        ;;
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        show_help
        exit 1
        ;;
esac