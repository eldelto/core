package main

import (
	"log"
	"net/url"
	"os"

	"github.com/eldelto/core/internal/personio"
)

func main() {
	url, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	client := personio.Client{Host: url}

	if err := client.Login(); err != nil {
		log.Fatal(err)
	}
}
