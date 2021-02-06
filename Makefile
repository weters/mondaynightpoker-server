.PHONY: test
test:
	golangci-lint run
	go run ./cmd/migrate
	go test -cover ./...

.PHONY: keys
keys: .keys/public.pem

.keys/public.pem: .keys/private.key
	openssl rsa -in .keys/private.key -pubout -out .keys/public.pem

.keys/private.key:
	mkdir -p .keys
	openssl genrsa -out .keys/private.key 2048

.PHONY: dev-database
dev-database:
	-docker run --name mondaynightpoker -e POSTGRES_HOST_AUTH_METHOD=trust -d -p 5432:5432 postgres:9.4
	go run ./cmd/migrate

clean:
	-docker rm -v -f mondaynightpoker
	rm -rf .keys/public.pem .keys/private.key