package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type Server struct {
	Port                   int    `yaml:"port" json:"port"`
	Address                string `yaml:"address" json:"address"`
	DownloadPath           string `yaml:"download_path" json:"download_path"`
	MaximumActiveDownloads int    `yaml:"maximum_active_downloads_per_domain" json:"maximum_active_downloads_per_domain"`
}

type DownloadProfile struct {
	Name    string   `yaml:"name" json:"name"`
	Command string   `yaml:"command" json:"command"`
	Args    []string `yaml:"args" json:"args"`
}

type UI struct {
	PopupWidth  int `yaml:"popup_width" json:"popup_width"`
	PopupHeight int `yaml:"popup_height" json:"popup_height"`
}

type Config struct {
	ConfigVersion    int               `yaml:"config_version" json:"config_version"`
	Server           Server            `yaml:"server" json:"server"`
	UI               UI                `yaml:"ui" json:"ui"`
	DownloadProfiles []DownloadProfile `yaml:"profiles" json:"profiles"`
}

// ConfigService is a struct to handle configuration requests, allowing for the
// location that config files are loaded to be customised.
type ConfigService struct {
	Config     *Config
	ConfigPath string
}

func (cs *ConfigService) LoadTestConfig() {
	cs.LoadDefaultConfig()
	cs.Config.DownloadProfiles = []DownloadProfile{{Name: "test profile", Command: "sleep", Args: []string{"5"}}}
}

func (cs *ConfigService) LoadDefaultConfig() {
	defaultConfig := Config{}
	stdProfile := DownloadProfile{Name: "standard video", Command: "youtube-dl", Args: []string{
		"--newline",
		"--write-info-json",
		"-f",
		"bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
	}}
	mp3Profile := DownloadProfile{Name: "standard mp3", Command: "youtube-dl", Args: []string{
		"--newline",
		"--write-info-json",
		"--extract-audio",
		"--audio-format", "mp3",
	}}

	defaultConfig.DownloadProfiles = append(defaultConfig.DownloadProfiles, stdProfile)
	defaultConfig.DownloadProfiles = append(defaultConfig.DownloadProfiles, mp3Profile)

	defaultConfig.Server.Port = 6123
	defaultConfig.Server.Address = "http://localhost:6123"
	defaultConfig.Server.DownloadPath = "./"

	defaultConfig.UI.PopupWidth = 500
	defaultConfig.UI.PopupHeight = 500

	defaultConfig.Server.MaximumActiveDownloads = 2

	defaultConfig.ConfigVersion = 2

	cs.Config = &defaultConfig

	return
}

func (c *Config) ProfileCalled(name string) *DownloadProfile {
	for _, p := range c.DownloadProfiles {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

func (c *Config) UpdateFromJSON(j []byte) error {
	newConfig := Config{}
	err := json.Unmarshal(j, &newConfig)
	if err != nil {
		log.Printf("Unmarshal error in config: %v", err)
		return err
	}

	// sanity checks
	if newConfig.UI.PopupHeight < 100 || newConfig.UI.PopupHeight > 2000 {
		return errors.New("invalid popup height - should be 100-2000")
	}
	if newConfig.UI.PopupWidth < 100 || newConfig.UI.PopupWidth > 2000 {
		return errors.New("invalid popup width - should be 100-2000")
	}

	// check listen port
	if newConfig.Server.Port < 1 || newConfig.Server.Port > 65535 {
		return errors.New("invalid server listen port")
	}

	// check download path
	fi, err := os.Stat(newConfig.Server.DownloadPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("path '%s' does not exist", newConfig.Server.DownloadPath)
	}
	if !fi.IsDir() {
		return fmt.Errorf("path '%s' is not a directory", newConfig.Server.DownloadPath)
	}

	if newConfig.Server.MaximumActiveDownloads < 0 {
		return fmt.Errorf("maximum active downloads can not be < 0")
	}

	// check profile name uniqueness
	for i, p1 := range newConfig.DownloadProfiles {
		for j, p2 := range newConfig.DownloadProfiles {
			if i != j && p1.Name == p2.Name {
				return fmt.Errorf("duplicate download profile name '%s'", p1.Name)
			}
		}
	}

	// remove leading/trailing spaces from args and commands and check for emptiness
	for i := range newConfig.DownloadProfiles {
		newConfig.DownloadProfiles[i].Name = strings.TrimSpace(newConfig.DownloadProfiles[i].Name)

		if newConfig.DownloadProfiles[i].Name == "" {
			return errors.New("profile name cannot be empty")
		}

		newConfig.DownloadProfiles[i].Command = strings.TrimSpace(newConfig.DownloadProfiles[i].Command)
		if newConfig.DownloadProfiles[i].Command == "" {
			return fmt.Errorf("command in profile '%s' cannot be empty", newConfig.DownloadProfiles[i].Name)
		}

		// check the args
		for j := range newConfig.DownloadProfiles[i].Args {
			newConfig.DownloadProfiles[i].Args[j] = strings.TrimSpace(newConfig.DownloadProfiles[i].Args[j])
			if newConfig.DownloadProfiles[i].Args[j] == "" {
				return fmt.Errorf("argument %d of profile '%s' is empty", j+1, newConfig.DownloadProfiles[i].Name)
			}
		}

		// check the command exists
		_, err := exec.LookPath(newConfig.DownloadProfiles[i].Command)
		if err != nil {
			return fmt.Errorf("Could not find %s on the path", newConfig.DownloadProfiles[i].Command)
		}
	}

	*c = newConfig
	return nil
}

// DetermineConfigDir determines where the config is (or should be) stored.
func (cs *ConfigService) DetermineConfigDir() {
	// check current directory first, for a file called gropple.yml
	_, err := os.Stat("gropple.yml")
	if err == nil {
		// exists in current directory, use that.
		cs.ConfigPath = "gropple.yml"
		return
	}

	// otherwise fall back to using the UserConfigDir
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("cannot find a directory to store config: %v", err)
	}
	appDir := "gropple"

	fullPath := dir + string(os.PathSeparator) + appDir
	_, err = os.Stat(fullPath)

	if os.IsNotExist(err) {
		err := os.Mkdir(fullPath, 0777)
		if err != nil {
			log.Fatalf("Could not create config dir '%s': %v", fullPath, err)
		}
	}

	fullFilename := fullPath + string(os.PathSeparator) + "config.yml"
	cs.ConfigPath = fullFilename
}

// ConfigFileExists checks if the config file already exists, and also checks
// if there is an error accessing it
func (cs *ConfigService) ConfigFileExists() (bool, error) {
	path := cs.ConfigPath
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("could not check if '%s' exists: %s", path, err)
	}
	if info.Size() == 0 {
		return false, errors.New("config file is 0 bytes")
	}
	return true, nil
}

// LoadConfig loads the configuration from disk, migrating and updating it to the
// latest version if needed.
func (cs *ConfigService) LoadConfig() error {
	path := cs.ConfigPath
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Could not read config '%s': %v", path, err)
	}
	c := Config{}
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return fmt.Errorf("Could not parse YAML config '%s': %v", path, err)
	}

	// do migrations
	configMigrated := false
	if c.ConfigVersion == 1 {
		c.Server.MaximumActiveDownloads = 2
		c.ConfigVersion = 2
		configMigrated = true
		log.Print("migrated config from version 1 => 2")
	}

	if configMigrated {
		log.Print("Writing new config after version migration")
		cs.WriteConfig()
	}

	cs.Config = &c

	return nil
}

// WriteConfig writes the in-memory config to disk.
func (cs *ConfigService) WriteConfig() {
	s, err := yaml.Marshal(cs.Config)
	if err != nil {
		panic(err)
	}

	path := cs.ConfigPath
	file, err := os.Create(
		path,
	)

	if err != nil {
		log.Fatalf("Could not open config file %s: %s", path, err)
	}
	defer file.Close()

	file.Write(s)
	file.Close()
}
