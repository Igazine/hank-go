# Hank for Go

Hank is a purely symbolic, instruction-oriented embeddable language designed to bring secure, dynamic automation to any host application. Built on a strict air-gapped execution model, Hank has zero built-in I/O, guaranteeing that scripts cannot access the filesystem, network, or OS without explicit delegation. This makes it the perfect predictable environment for game scripting, microservice orchestration, and user-facing plugin systems. With a highly readable, keyword-less syntax and universal cross-platform parity, Hank seamlessly bridges the gap between static configuration files and complex general-purpose programming.

This repository provides the official Go implementation of the Hank language. It is a reusable package for embedding the Hank interpreter and runner into any Go application.

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
