FROM golang:1.24 AS build

RUN mkdir /src
WORKDIR /src
COPY . /src

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o application -ldflags  "-X main.CommitID=$(git log -1 --format=%H) -X main.BuildBranch=$(git rev-parse --abbrev-ref HEAD) -X main.BuildTimestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)" ./cmd/loadmonitor/main.go

FROM scratch
COPY --from=build /src/application /application
COPY ./static /static
ENTRYPOINT ["/application"]