build:
	docker-compose build

up: build
	docker-compose up

silent:
	docker-compose build > build.out &

service: silent
	nohup docker-compose up > log.out &

go:
	go build .