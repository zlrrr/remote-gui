.PHONY: certs test build docker compose-up lint clean

certs:
	bash scripts/gen-certs.sh

test:
	cd remote-executor && go test ./... -race -v
	cd gui             && go test ./... -race -v

build:
	cd remote-executor && go build -o dist/remote-executor ./cmd/executor/
	cd gui             && go build -o dist/gui             ./cmd/gui/

docker:
	docker build -t remote-executor -f remote-executor/deployments/Dockerfile remote-executor/
	docker build -t remote-gui      -f gui/deployments/Dockerfile              gui/

compose-up:
	docker-compose up -d

lint:
	cd remote-executor && golangci-lint run ./...
	cd gui             && golangci-lint run ./...

clean:
	rm -rf remote-executor/dist gui/dist certs/*.key certs/*.crt certs/*.csr
