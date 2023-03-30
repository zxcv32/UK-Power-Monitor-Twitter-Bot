FROM golang:1.20.2-bullseye AS build
WORKDIR /bot/
COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify
COPY src ./src/
RUN CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -tags=nomsgpack -a -o app ./src

FROM golang:1.20.2-bullseye
WORKDIR /app
ENV GIN_MODE=release
COPY --from=build /bot/app ./app
CMD ["./app"]
