package main

import (
	"flag"
	"fmt"

	//"os"

	"github.com/a-schus/REST-API/internal/app/apiserver"
	"github.com/a-schus/REST-API/internal/app/store"
)

func main() {
	ip := flag.String("ip", "localhost:8080", "IP")

	flag.Parse()

	db := store.Store{}

	if db.Open() != nil {
		return
	}
	defer db.Close()

	server := apiserver.New(*ip, &db)
	server.Start()

	fmt.Println("!Ok")
}
