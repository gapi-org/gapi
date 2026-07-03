# Installation

Gapi is distributed as a normal Go module.

## Add Gapi To An App

`go get` adds dependencies to an existing module, so start by creating or entering an app with a `go.mod` file:

```bash
mkdir hello-api
cd hello-api
go mod init hello-api
go get github.com/gapi-org/gapi@latest
```

Then import the public framework package:

```go
import "github.com/gapi-org/gapi"
```

Optional packages:

```go
import "github.com/gapi-org/gapi/middleware"
import gapitest "github.com/gapi-org/gapi/testing"
```

This matches the usual Go framework pattern:

```go
import "github.com/gin-gonic/gin"
import "github.com/labstack/echo/v4"
import "github.com/go-chi/chi/v5"
```

## Install The CLI

The CLI is installed with `go install`, which can be run from any directory:

```bash
go install github.com/gapi-org/gapi/cmd/gapi@latest
```

Go installs command binaries into `$(go env GOPATH)/bin` unless `GOBIN` is set. If `gapi` is not found after installation, add that directory to your shell path:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Then scaffold an app:

```bash
gapi new hello-api
cd hello-api
go mod tidy
go run .
```

## Pin Versions

For production apps, pin a released version:

```bash
go get github.com/gapi-org/gapi@v0.1.0
go install github.com/gapi-org/gapi/cmd/gapi@v0.1.0
```

For unreleased framework development, use a Go workspace or `replace` directive rather than editing imports:

```bash
go work init ./gapi ./my-api
```

The import path stays `github.com/gapi-org/gapi`; the workspace decides whether Go reads it from disk or downloads it from the module proxy.
