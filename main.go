package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/tardisx/gropple/config"
	"github.com/tardisx/gropple/download"
	"github.com/tardisx/gropple/version"
	"github.com/tardisx/gropple/web"
)

func main() {
	versionInfo := &version.Manager{
		VersionInfo: version.Info{CurrentVersion: "v1.1.3"},
	}
	log.Printf("Starting gropple %s - https://github.com/tardisx/gropple", versionInfo.GetInfo().CurrentVersion)

	var configPath string
	flag.StringVar(&configPath, "config-path", "", "path to config file")

	flag.Parse()

	configService := &config.ConfigService{}
	if configPath != "" {
		configService.ConfigPath = configPath
	} else {
		configService.DetermineConfigDir()
	}

	exists, err := configService.ConfigFileExists()
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		log.Print("No config file - creating default config")
		configService.LoadDefaultConfig()
		configService.WriteConfig()
		log.Printf("Configuration written to %s", configService.ConfigPath)
	} else {
		err := configService.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Configuration loaded from %s", configService.ConfigPath)
	}

	// create the download manager
	downloadManager := &download.Manager{MaxPerDomain: configService.Config.Server.MaximumActiveDownloads}

	// create the web handlers
	r := web.CreateRoutes(configService, downloadManager, versionInfo)

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf(":%d", configService.Config.Server.Port),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}

	// check for a new version every 4 hours
	go func() {
		for {
			err := versionInfo.UpdateGitHubVersion()
			if err != nil {
				log.Printf("could not get version info: %s", err)
			}
			time.Sleep(time.Hour * 4)
		}
	}()

	// start downloading queued downloads when slots available, and clean up
	// old entries
	go downloadManager.ManageQueue()

	// add testdata if compiled with the '-tags testdata' flag
	downloadManager.AddStressTestData(configService)

	log.Printf("Visit %s for details on installing the bookmarklet and to check status", configService.Config.Server.Address)
	log.Fatal(srv.ListenAndServe())

}
