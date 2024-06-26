package main

import (
	"flag"
	"fmt"

	"github.com/a-schus/REST-API/internal/app/apiserver"
	"github.com/a-schus/REST-API/internal/app/store"
)

func main() {
	ip := flag.String("ip", "localhost:8080", "IP")
	user := flag.String("n", "", "User name")
	pass := flag.String("p", "", "User password")
	host := flag.String("dbip", "localhost", "Database server IP")

	flag.Parse()

	conf := store.DBConf{
		User: *user,
		Pass: *pass,
		Host: *host,
		Name: "restapi_dev",
	}

	db := store.Store{}

	if db.Open(conf) != nil {
		return
	}
	defer db.Close()

	server := apiserver.New(*ip, &db)
	server.Start()

	fmt.Println("!Ok")
}
