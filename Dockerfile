FROM golang:1.18 as builder
ENV CGO_ENABLED=0
WORKDIR /go/src/
# Workaround for Private Github REPOs
ARG GITHUB_TOKEN
RUN go env -w GOPRIVATE="*"
RUN git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -v -o /fnwrapper-server/fnwrapper-server

FROM alpine:3.17
COPY --from=builder /fnwrapper-server/* /fnwrapper-server/