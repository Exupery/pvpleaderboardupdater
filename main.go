package main

import (
	"log"
	"os"
)

var logger *log.Logger = log.New(os.Stdout, "main|", log.Ltime|log.Lmicroseconds)

func main() {
	logger.Println("PvPLeaderBoard Starting")

	logger.Println("PvPLeaderBoard Complete")
}
