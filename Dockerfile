FROM golang:alpine as build

WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY . .

RUN go build -o announcer

FROM alpine
WORKDIR /announcer
COPY --from=build /app/announcer /bin/announcer

ENTRYPOINT [ "/bin/announcer" ]