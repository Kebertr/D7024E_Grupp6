#!/bin/bash
go test -coverprofile=coverage.out ./internal/kademlia/... -count=1
go tool cover -html=coverage.out
# Source: https://git.ludd.ltu.se/antonn 