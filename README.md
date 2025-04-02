# Outline → Zulip Webhook Bridge

This is a lightweight webhook bridge that listens for [Outline](https://www.getoutline.com/) webhook requests and sends
formatted notifications to a [Zulip](https://zulip.com/) stream for each event it receives.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## Features

- Secure HMAC validation of Outline webhook requests
- Recognizes and formats `create`, `update`, `delete`, and `title_change` events, additional events can be easily added
- Sends small Markdown formatted messages to a Zulip stream and topic, including a snippet of the document text and a link directly to the document
- Written in idiomatic Go with no external dependencies making it extremely lightweight
- Easy to modify for your own needs

---

## Getting Started

### 1. Clone the repo

```bash
git clone https://github.com/YOUR_USERNAME/outline-zulip-webhook.git
cd outline-zulip-webhook
```

### 2. Build and run the app

```bash
make build
make run
```

or

```bash
go build -o outline-zulip-webhook
```

A dockerfile is included for development, and can be built and run using the `make run` command.

A dockerfile for building images to publish into a repository is also included as `release.dockerfile` and can be built using the following commands:

```bash
export GIT_TAG=$(git describe --tags --exact-match)
export DOCKER_REGISTRY=your.docker.registry
export APP_NAME=outline-zulip-webhook

docker build -f release.dockerfile \
    -t $(DOCKER_REGISTRY)/$(APP_NAME):$(GIT_TAG) \
    -t $(DOCKER_REGISTRY)/$(APP_NAME):latest .
docker push $(DOCKER_REGISTRY)/$(APP_NAME):$(GIT_TAG)
docker push $(DOCKER_REGISTRY)/$(APP_NAME):latest
```

#### Important Note

This bridge does process any inbound TLS or SSL connections, so it needs to be run behind a reverse proxy such as Caddy or Traefik.

The following environment variables will need to be set before running the bridge. It was intended to be run in Docker or Kubernetes, so this should be done in the respective configuration for the deployment method used:

| Variable               | Description                                                                                                                                                                                      |
|------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ZULIP_WEBHOOK_URL      | The URL of the Zulip webhook endpoint, this should be in the format https://user:pass@url/api/v1/messages, i.e. 'https://user-bot@zulip.example.com:password@zulip.example.com/api/v1/messages'  |
| ZULIP_STREAM           | The name of the Zulip stream to send notifications to, i.e. 'general'                                                                                                                            |
| ZULIP_TOPIC            | The name of the Zulip topic to send notifications to, i.e. 'Wiki Update'                                                                                                                         |
| OUTLINE_WEBHOOK_SECRET | The secret key used to validate webhook requests from Outline, this is obtained from Outline's webhook configuration                                                                             |
| OUTLINE_BASE_URL       | The base URL of your Outline installation without the trailing slash, i.e. 'https://outline.example.com'                                                                                         |
| PORT                   | Optional, the default port is 8484                                                                                                                                                               |


### 3. Zulip Setup

Add a new Zulip bot from Personal Settings, Bots. It should be an incoming webhook bot. This should generate the "Bot Email" and the API Key for the bot. These are used for the user:pass portion of the `ZULIP_WEBHOOK_URL`.

The stream should be an existing stream, though the topic doesn't need to be existing yet. It will be created when the first post is made.

### 4. Outline Setup

Add a new outbound webhook from the preferences option on your Outline instance in the Webhooks section.

When selecting the URL, it will only allow https connections, so the bridge will need to be behind a reverse proxy that terminates TLS.

The full URL should be something like `https://outline-zulip.example.com/outline-webhook` where `outline-zulip.example.com` is the hostname of the bridge as configured in the reverse proxy.

Make note of the signing secret and add that into the `OUTLINE_WEBHOOK_SECRET` environment variable. This is used to validate the webhook requests from Outline, any messages that fail this validation will be ignored.

Hopefully at some point Outline will allow non SSL http connections for webhooks (which seems unlikely), but that would allow the bridge to run in Kubernetes or Docker deployments without requiring an ingress from the public internet.

### 5. CI/CD

There is also a `.gitea/workflows/release.yml` file that will build a docker image and push it to a docker registry when a new release is created on Gitea.

It will require a `REGISTRY_USERNAME` and `REGISTRY_PASSWORD` secret in Gitea's Actions - Secrets, and the `REGISTRY` and `IMAGE_NAME` environment variables to be set.

For example:

| Variable   | Example                      |
|------------|------------------------------|
| REGISTRY   | gitea.example.com            |
| IMAGE_NAME | orgname/outline-zulip-bridge |
