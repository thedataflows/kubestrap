# Kubestrap
>
> Note: this software is an alpha, work in progress.

Toolbox acting as a wrapper over some known utilities, to ease bootstrapping and maintenance of Kubernetes clusters and the apps deployed on them, in a [GitOps manner](https://www.weave.works/technologies/gitops/).

## Features

## Setup

## Run It 🏃

`go run main.go`

## Usage

- `kubestrap`

    ```properties

    ```

- `kubestrap sample-command -h`

    ```properties

    ```

## Configure It ☑️

- See [sample/myconfig.yaml](./sample/myconfig.yaml) for config file
- All parameters can be set via flags or env as well: `MYPREFIX_<subcommand>_<flag>`, example: `MYPREFIX_SAMPLE_COMMAND_FLAG1=1122334455`

## Test It 🧪

Test for coverage and race conditions

`make coverage`

## Lint It 👕

`make pre-commit run --all-files --show-diff-on-failure`

## Roadmap

- [ ] ?

## Development

### Build

- Preferably: `goreleaser build --clean --single-target` or
- `make build` or
- `scripts/local-build.sh` (deprecated)
