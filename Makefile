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
	-docker run --name mondaynightpoker -e POSTGRES_HOST_AUTH_METHOD=trust -d -p 5432:5432 postgres:16
	go run ./cmd/migrate

clean:
	-docker rm -v -f mondaynightpoker
	rm -rf .keys/public.pem .keys/private.key

assets/uml_room.png:
	goplantuml -aggregate-private-members \
			   -show-aggregations \
			   -hide-private-members \
			   pkg/room pkg/playable \
		| java -jar third_party/plantuml/plantuml.jar -pipe > assets/uml_room.png

assets/uml_playable.png:
	goplantuml -aggregate-private-members \
			   -show-aggregations \
			   -hide-private-members \
			   pkg/playable \
			   pkg/playable/poker/handanalyzer \
			   pkg/playable/poker/sevencard \
		| java -jar third_party/plantuml/plantuml.jar -pipe > assets/uml_playable.png

assets/uml_deck.png:
	goplantuml -aggregate-private-members \
			   -show-aggregations \
			   -hide-private-members \
			   pkg/deck \
		| java -jar third_party/plantuml/plantuml.jar -pipe > assets/uml_deck.png

uml: assets/uml_playable.png assets/uml_room.png assets/uml_deck.png