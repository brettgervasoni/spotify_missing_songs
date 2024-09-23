package main

import (
	"encoding/json"
	"os"

	"golang.org/x/oauth2"
)

// authentication related functions for spotify

func saveTokenToFile(token *oauth2.Token) error {
	file, err := os.Create(tokenFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(token)
}

func readTokenFromFile() (*oauth2.Token, error) {
	file, err := os.Open(tokenFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var token oauth2.Token
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&token)
	return &token, err
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
