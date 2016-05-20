# https

A little https proxy with no frills for local development. Generates a self-signed certificate on-demand.

## installation

## via homebrew

```bash
$ brew install mattrobenolt/stuff/https
```

## via curl

```bash
$ curl -sSL https://github.com/mattrobenolt/https/releases/download/0.1.0/https-darwin-amd64 > /usr/local/bin/https && chmod +x /usr/local/bin/https
```

### via go

```bash
$ go get github.com/mattrobenolt/https
```

## usage

```bash
$ https 8080
$ https -host=example.dev 9000
$ https -listen=:8888
```
