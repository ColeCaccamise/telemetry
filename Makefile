.PHONY: build run dev clean prod

build:
	mkdir -p bin
	go build -o bin/telemetry server.go

run:
	go run server.go

dev: build run

prod: build
	./bin/telemetry

clean:
	rm -rf bin
