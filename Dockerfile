FROM golang:1.18 AS build
WORKDIR /bot
COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -tags=nomsgpack -a -o app .

FROM golang:1.18
WORKDIR /app
COPY --from=build /bot/app ./app
CMD ["./app"]
