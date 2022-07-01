# UK-Power-Monitor-Twitter-Bot

Publish UK Power Monitor's stats on Twitter

Source data is generated from [UK-Power-Monitor](https://github.com/zxcV32/UK-Power-Monitor)

[![Bot Twitter URL](https://img.shields.io/twitter/url/https/twitter.com/UKPowerMonitor.svg?style=social&label=Follow%20%40UKPowerMonitor)](https://twitter.com/UKPowerMonitor)
[![Creator Twitter URL](https://img.shields.io/twitter/url/https/twitter.com/i14a23h19a.svg?style=social&label=Follow%20%40i14a23h19a)](https://twitter.com/i14a23h19a)

## Build And Deploy

### Local

1. Create [Twitter Project](https://developer.twitter.com/en/docs/projects/overview)
   with `OAuth 1.0a` and apply for
   [Elevated Access Level](https://developer.twitter.com/en/docs/twitter-api/getting-started/about-twitter-api)
   for [Manage Tweet](https://developer.twitter.com/en/docs/twitter-api/tweets/manage-tweets/introduction)
   permissions [here](https://developer.twitter.com/en/portal/products/elevated)
1. Clone this project
1. Create .env and populate the variables
   
   `cp .env.template .env`
1. Issue command `go run main.go` from root of the project

### Docker [Dev]
1. `docker-compose up -d --build`

### Docker [Prod]
1. Build and push Docker Image to registry.
   Example: 
   ```
   $ docker build . -t zxcv32/uk-power-monitor-twitter-bot:main
   $ docker push zxcv32/uk-power-monitor-twitter-bot:main
   ```
1. Create .env file in the Docker host with all the required configurations
1. Copy `docker-compose.prod.yml` to Docker host
1. `docker-compose -f docker-compose.prod.yml pull`
1. `docker-compose -f docker-compose.prod.yml up -d`
