#!/bin/env bash

PACKAGE_NAME=kubestrap
PACKAGE_VERSION="${PACKAGE_VERSION:-v0.0.0}"
PACKAGE_BIN=${PACKAGE_NAME}
PACKAGE_REPO=https://github.com/thedataflows/kubestrap

SCRIPT_DIR=$(readlink -f "$0")
SCRIPT_DIR="${SCRIPT_DIR%/*}"

if [[ "$OS" == "Windows_NT" ]]; then
    [[ $(type -p cygpath) ]] && SCRIPT_DIR=$(cygpath -u "$SCRIPT_DIR")
    PACKAGE_BIN=${PACKAGE_NAME}.exe
fi

DEST="$SCRIPT_DIR/bin/$PACKAGE_NAME/$PACKAGE_VERSION"
PATH="$DEST:$PATH"

## Install the correct version of the package
if [[ ! $(type -p $PACKAGE_BIN) || "$($PACKAGE_BIN version)" != "$PACKAGE_VERSION" ]]; then
    [[ ! -d "$DEST" ]] && mkdir -p "$DEST"
    if [[ "$OS" == "Windows_NT" ]]; then
        set -x
        curl -L "$PACKAGE_REPO/releases/download/${PACKAGE_VERSION}/${PACKAGE_NAME}_${PACKAGE_VERSION}_windows_amd64.zip" > "$DEST/$PACKAGE_NAME.zip" && \
            unzip -u "$DEST/$PACKAGE_NAME.zip" -d "$DEST" && \
            rm -vf "$DEST/$PACKAGE_NAME.zip"
        { set +x; } 2>/dev/null
    else
        set -x
        curl -L "$PACKAGE_REPO/releases/download/${PACKAGE_VERSION}/${PACKAGE_NAME}_${PACKAGE_VERSION}_linux_amd64.tar.gz" | tar --directory "$DEST" -xzvf - && \
            chmod -v +x "$DEST/$PACKAGE_NAME"
        { set +x; } 2>/dev/null
    fi
fi

CONFIG_COMMAND=
if [[ -z "$CLUSTER_CONFIG" && -n "$KUBERNETES_CLUSTER_CONTEXT" ]]; then
    CLUSTER_CONFIG="$SCRIPT_DIR/kubestrap-$KUBERNETES_CLUSTER_CONTEXT.yaml"
    if [[ -f "$CLUSTER_CONFIG" ]]; then
      CONFIG_COMMAND="--config $CLUSTER_CONFIG"
    fi
fi

set -x
$PACKAGE_BIN --config $SCRIPT_DIR/kubestrap-defaults.yaml $CONFIG_COMMAND "$@"
{ set +x; } 2>/dev/null
