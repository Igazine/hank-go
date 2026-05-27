# Hank for Go

A Go implementation of the Hank language.

This repository provides a reusable Go package (`github.com/Igazine/hank-go`) for embedding the Hank interpreter and runner into any Go application.

## Installation

```bash
go get github.com/Igazine/hank-go
```

## Features

- **Standard Library**: Full support for the official Hank Standard Library (`math`, `str`, `arr`, `obj`, `logic`, `json`, `log`, `runtime`, `env`).
- **High Performance**: Pure Go implementation optimized for orchestration tasks.
- **Embedded Examples**: Includes a reference runner implementation.

## Testing & Examples

An example CLI runner is included in `examples/runner`. Note that the runner requires the universal conformance suite located in the `hank` submodule.

To fetch submodules after cloning:

```bash
git submodule update --init --recursive
```

To run the conformance tests:

```bash
cd examples/runner
go run main.go
```

## Project Links

- **Hank Core Repo**: [Igazine/hank](https://github.com/Igazine/hank)
- **Official Documentation**: [https://igazine.github.io/hank/](https://igazine.github.io/hank/)

## License

This project is licensed under the MIT License.
