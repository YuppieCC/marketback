.PHONY: test test-integration

test:
	go test -v ./...

test-integration:
	docker-compose -f docker-compose.test.yml up -d
	sleep 5
	go test -v ./test/integration/...
	docker-compose -f docker-compose.test.yml down 