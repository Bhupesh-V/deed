package main

import (
	"context"
	"deed/internal/deed"
)

func main() {
	ctx := context.Background()
	err := deed.New("postgres://postgres:my_secure_password@127.0.0.1:5433/postgres").Start(ctx)
	if err != nil {
		panic(err)
	}
}
