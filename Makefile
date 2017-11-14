build:
	docker-compose build

silent:
	docker-compose build > build.out &

up: build
	docker-compose up

service: build
	nohup docker-compose up > log.out &

go:
	go build .