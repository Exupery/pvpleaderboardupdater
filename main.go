package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var logger *log.Logger = log.New(os.Stdout, "main|", log.Ltime|log.Lmicroseconds)
const uriBase string = "https://us.battle.net/api/wow/"

func main() {
	logger.Println("PvPLeaderBoard Starting")
	getLeaderboard()
	logger.Println("PvPLeaderBoard Complete")
}

func get(path string) *string {
	const errPrefix string = "[ERROR] GET failed: "
	resp, err := http.Get(uriBase + path)

	if err != nil {
		logger.Println(errPrefix + err.Error())
		return nil
	}
	if resp.StatusCode != 200 {
		logger.Println(errPrefix + resp.Status)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		logger.Println(errPrefix + err.Error())
		return nil
	}

	logger.Printf("%s returned %v bytes", path, len(body))
	str := string(body[:])
	return &str
}

func getLeaderboard() {
	s := get("leaderboard/2v2")
	logger.Println(len(*s))	// TODO DELME
}
