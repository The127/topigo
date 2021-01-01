FROM golang:alpine as builder
RUN apk --no-cache add build-base git bzr mercurial gcc make
ADD . /src
RUN cd /src && make build

FROM alpine
WORKDIR /app
COPY --from=builder /src/topigo /app/
ENTRYPOINT ./topigo