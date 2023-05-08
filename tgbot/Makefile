all:
	sudo docker compose up -d --build

ls:
	sudo docker container ls -a

docker_start:
	sudo systemctl start docker

mongosh:
	sudo mongosh 172.20.0.2:27017

logs:
	sudo docker container logs -f tgbot-app-1

stop:
	sudo docker container stop tgbot-app-1
	sudo docker container stop tgbot-mongodb-1

linter:
	golangci-lint run ./...
