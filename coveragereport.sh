#!/bin/bash
go test -race -coverprofile=coverage.out ./internal/... -count=1
grep -v '\.pb\.go:' coverage.out > coverage.filtered.out

go tool cover -func=coverage.filtered.out
go tool cover -html=coverage.filtered.out -o coverage.html
# Thanks to https://git.ludd.ltu.se/antonn 