.PHONY: up
up:
	docker-compose build app
	docker-compose up