FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /gated ./cmd/gated

FROM alpine:3.19
COPY --from=build /gated /usr/local/bin/gated
EXPOSE 8090
ENTRYPOINT ["gated"]
