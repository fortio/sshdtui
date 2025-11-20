[![GoDoc](https://godoc.org/fortio.org/sshd?status.svg)](https://pkg.go.dev/fortio.org/sshd)
[![Go Report Card](https://goreportcard.com/badge/fortio.org/sshd)](https://goreportcard.com/report/fortio.org/sshd)
[![GitHub Release](https://img.shields.io/github/release/fortio/sshd.svg?style=flat)](https://github.com/fortio/sshd/releases/)
[![CI Checks](https://github.com/fortio/sshd/actions/workflows/include.yml/badge.svg)](https://github.com/fortio/sshd/actions/workflows/include.yml)
[![codecov](https://codecov.io/github/fortio/sshd/graph/badge.svg?token=Yx6QaeQr1b)](https://codecov.io/github/fortio/sshd)

# sshd

Ansipixels sshd demoes menu server

## Install
You can get the binary from [releases](https://github.com/fortio/sshd/releases)

Or just run
```
CGO_ENABLED=0 go install fortio.org/sshd@latest  # to install (in ~/go/bin typically) or just
CGO_ENABLED=0 go run fortio.org/sshd@latest  # to run without install
```

or
```
brew install fortio/tap/sshd
```

or
```
docker run -ti fortio/sshd
```


## Usage

```
sshd help

flags:
```
