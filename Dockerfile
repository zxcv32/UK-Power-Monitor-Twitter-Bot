FROM golang:1.18.2-stretch AS build
WORKDIR /bot/
COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify
COPY src ./src/
RUN CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -tags=nomsgpack -a -o app ./src

FROM golang:1.18.2-stretch
WORKDIR /app
COPY --from=build /bot/app ./app
CMD ["./app"]
