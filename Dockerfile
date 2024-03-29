FROM alpine:3.17

ARG RELEASE=latest

WORKDIR /app

RUN apk add curl tar gunzip && \
  curl https://github.com/thedataflows/kubestrap/releases/download/${RELEASE}/kubestrap_${RELEASE}_linux_amd64.tar.gz | tar -xzvf -

USER nonroot:nonroot

ENTRYPOINT ["/app/kubestrap", "sample-command"]
