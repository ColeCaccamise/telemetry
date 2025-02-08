.PHONY: build run dev clean prod docker

build:
	mkdir -p bin
	go build -o bin/telemetry server.go

run:
	go run server.go

dev: build run

prod: build
	./bin/telemetry

docker:
	docker build -t telemetry .
	docker run -p 1323:1323 telemetry

clean:
	rm -rf bin
