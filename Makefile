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
	@mkdir -p .keys
	openssl genrsa -out .keys/private.key 2048
