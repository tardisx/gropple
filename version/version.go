// Package version deals with versioning of the software
package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/mod/semver"
)

type Info struct {
	CurrentVersion       string `json:"current_version"`
	GithubVersion        string `json:"github_version"`
	UpgradeAvailable     bool   `json:"upgrade_available"`
	GithubVersionFetched bool   `json:"-"`
}

func (i *Info) UpdateGitHubVersion() error {
	i.GithubVersionFetched = false
	versionUrl := "https://api.github.com/repos/tardisx/gropple/releases"
	resp, err := http.Get(versionUrl)
	if err != nil {
		log.Fatal("Error getting response. ", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %v", err)
	}

	type release struct {
		HTMLUrl string `json:"html_url"`
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}

	var releases []release

	err = json.Unmarshal(b, &releases)
	if err != nil {
		return fmt.Errorf("failed to read unmarshal: %v", err)
	}
	if len(releases) == 0 {
		log.Printf("found no releases on github?")
		return errors.New("no releases found")
	}

	i.GithubVersion = releases[0].Name

	i.GithubVersionFetched = true
	i.UpgradeAvailable = i.canUpgrade()
	return nil
}

func (i *Info) canUpgrade() bool {
	if !i.GithubVersionFetched {
		return false
	}

	log.Printf("We are %s, github is %s", i.CurrentVersion, i.GithubVersion)

	if !semver.IsValid(i.CurrentVersion) {
		log.Fatalf("current version %s is invalid", i.CurrentVersion)
	}

	if !semver.IsValid(i.GithubVersion) {
		log.Fatalf("github version %s is invalid", i.GithubVersion)
	}

	if semver.Compare(i.CurrentVersion, i.GithubVersion) == -1 {
		return true
	}
	return false
}
