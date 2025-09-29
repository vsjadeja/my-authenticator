#!/bin/bash

# Build and run the TOTP Authenticator application with system libraries
export CGO_LDFLAGS="-L/usr/lib/x86_64-linux-gnu"
export CGO_CPPFLAGS="-I/usr/include"

go build -o my-authenticator main.go
./my-authenticator