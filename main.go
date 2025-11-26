package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	steam "github.com/Philipp15b/go-steamapi"
)

func main() {
	apiKey := os.Getenv("STEAM_API_KEY")
	if apiKey == "" {
		log.Fatal("STEAM_API_KEY environment variable not set")
	}

	client := &http.Client{}
	api := steam.New(apiKey, client)

	apps, err := api.ISteamApps().GetAppList()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("List of Steam Apps:")
	for _, app := range apps.Apps {
		fmt.Printf("%d: %s\n", app.AppID, app.Name)
	}
}