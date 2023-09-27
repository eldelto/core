package main

import (
	"fmt"
	"log"
	"os"

	"github.com/eldelto/core/internal/diatom"
)

func usage() {
	fmt.Println("TODO: Usage message")
}

func main() {
	if len(os.Args) != 2 {
		usage()
		log.Fatal("expected single argument")
	}

	dasmPath := os.Args[1]
	file, err := os.Open(dasmPath)
	if err != nil {
		log.Fatalf("failed to open file at '%s': %v", dasmPath, err)
	}
	defer file.Close()

	parser := diatom.NewParser(file)
	for {
		token, err := parser.Token()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d: '%s'\n", parser.LineNumber, token)
		parser.Consume()
	}
}
