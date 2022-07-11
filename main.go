package main

import card "github.com/go-funcards/card-service/cmd"

//go:generate protoc -I proto --go_out=./proto/v1 --go-grpc_out=./proto/v1 proto/v1/card.proto

func main() {
	card.Execute()
}
