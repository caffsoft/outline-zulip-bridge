APP_NAME=outline-zulip-bridge
DOCKER_REGISTRY=git.caffsoft.dev/vueterix
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null || echo latest)

## run: Runs the app using the dev Dockerfile
run: build
	docker build -f dockerfile -t $(APP_NAME) .
	docker run --rm -it \
		--env-file .env.dev \
		-p 8484:8484 \
		$(APP_NAME)

## build: Builds the Go binary
build:
	CGO_ENABLED=0 go build -o bin/$(APP_NAME) .

release:
	docker build -f release.dockerfile \
		-t $(DOCKER_REGISTRY)/$(APP_NAME):$(GIT_TAG) \
		-t $(DOCKER_REGISTRY)/$(APP_NAME):latest .
	docker push $(DOCKER_REGISTRY)/$(APP_NAME):$(GIT_TAG)
	docker push $(DOCKER_REGISTRY)/$(APP_NAME):latest
