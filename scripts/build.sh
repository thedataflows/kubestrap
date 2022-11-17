#!/usr/bin/env bash

set -e -o pipefail

PROJECT_PATH=$(readlink -f ${0%/*}/..)

PROJECT_MODULE=$(grep -oP '(?<=module ).+' $PROJECT_PATH/go.mod)
PROJECT_NAME=${PROJECT_MODULE##*/}
EXT=
[[ "$OS" == "Windows_NT" || "$GOOS" == "windows" ]] && EXT='.exe'

(
  cd ${0%/*}/../
  # TODO improve tag extraction
  TAG=$(git tag --sort=-version:refname | head -n 1)
  [[ -z "$TAG" ]] && TAG=dev
  go build \
    -ldflags "-s -w -X '$PROJECT_MODULE/cmd.version=$TAG'" \
    -trimpath \
    -o $PROJECT_PATH/bin/$PROJECT_NAME$EXT \
    main.go
)
