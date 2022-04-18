// Package version deals with versioning of the software
package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/mod/semver"
)

type Info struct {
	CurrentVersion       string `json:"current_version"`
	GithubVersion        string `json:"github_version"`
	UpgradeAvailable     bool   `json:"upgrade_available"`
	GithubVersionFetched bool   `json:"-"`
}

type Manager struct {
	VersionInfo Info
	lock        sync.Mutex
}

func (m *Manager) GetInfo() Info {
	// log.Print("getting info... b4 lock")
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.VersionInfo
}

func (m *Manager) UpdateGitHubVersion() error {
	m.lock.Lock()
	m.VersionInfo.GithubVersionFetched = false
	m.lock.Unlock()

	versionUrl := "https://api.github.com/repos/tardisx/gropple/releases"
	resp, err := http.Get(versionUrl)
	if err != nil {
		log.Printf("Error getting response: %v", err)
		return err
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

	m.lock.Lock()

	defer m.lock.Unlock()
	m.VersionInfo.GithubVersion = releases[0].Name
	m.VersionInfo.GithubVersionFetched = true
	m.VersionInfo.UpgradeAvailable = m.canUpgrade()

	return nil
}

func (m *Manager) canUpgrade() bool {
	if !m.VersionInfo.GithubVersionFetched {
		return false
	}

	if !semver.IsValid(m.VersionInfo.CurrentVersion) {
		log.Printf("current version %s is invalid", m.VersionInfo.CurrentVersion)
	}

	if !semver.IsValid(m.VersionInfo.GithubVersion) {
		log.Printf("github version %s is invalid", m.VersionInfo.GithubVersion)
	}

	if semver.Compare(m.VersionInfo.CurrentVersion, m.VersionInfo.GithubVersion) == -1 {
		return true
	}
	return false
}
