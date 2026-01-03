package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/krau/SaveAny-Bot/pkg/aria2"
)

func main() {
	// Create aria2 client
	client, err := aria2.NewClient("http://localhost:6800/jsonrpc", "")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Get aria2 version
	version, err := client.GetVersion(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("aria2 version: %s\n", version.Version)
	fmt.Printf("Enabled features: %v\n", version.EnabledFeatures)

	// Add a download
	uris := []string{"https://example.com/file.zip"}
	options := aria2.Options{
		"dir": "/downloads",
	}

	gid, err := client.AddURI(ctx, uris, options)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Download started with GID: %s\n", gid)

	// Monitor download progress
	for {
		status, err := client.TellStatus(ctx, gid)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Status: %s, Progress: %s/%s bytes, Speed: %s bytes/s\n",
			status.Status,
			status.CompletedLength,
			status.TotalLength,
			status.DownloadSpeed,
		)

		if status.IsDownloadComplete() {
			fmt.Println("Download completed!")
			break
		}

		if status.IsDownloadError() {
			fmt.Printf("Download error: %s\n", status.ErrorMessage)
			break
		}

		time.Sleep(1 * time.Second)
	}

	// Get global statistics
	stat, err := client.GetGlobalStat(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Global stats - Download speed: %s, Active: %s, Waiting: %s\n",
		stat.DownloadSpeed,
		stat.NumActive,
		stat.NumWaiting,
	)

	// List active downloads
	activeDownloads, err := client.TellActive(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Active downloads: %d\n", len(activeDownloads))
	for _, download := range activeDownloads {
		fmt.Printf("  GID: %s, Status: %s\n", download.GID, download.Status)
	}

	// Example with context timeout
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.TellStatus(ctxWithTimeout, gid)
	if err != nil {
		log.Printf("Request failed: %v\n", err)
	}
}
