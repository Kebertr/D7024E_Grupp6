#!/bin/bash
go test \
  -coverpkg=github.com/RasmusKebert/D7024E_Grupp6/internal/kademlia \
  -coverprofile=coverage.out \
  ./internal/kademlia/test \
  -count=1
go tool cover -html=coverage.out
# Thanks to https://git.ludd.ltu.se/antonn 