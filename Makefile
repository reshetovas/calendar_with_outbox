.PHONY: run rebuild stop logs clean

run:
	docker-compose up -d

rebuild:
	docker-compose up -d --build

stop:
	docker-compose down

clean:
	docker-compose down -v

logs:
	docker-compose logs -f
