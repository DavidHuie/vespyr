package main

import (
	"log"
	"os"

	"github.com/DavidHuie/vespyr/pkg/vespyr"
)

func main() {
	if err := vespyr.RootCmd.Execute(); err != nil {
		log.Printf("error starting vespyr: %s", err)
		os.Exit(1)
	}
}
