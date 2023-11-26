package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type Server struct {
	Port                   int    `yaml:"port" json:"port"`
	Address                string `yaml:"address" json:"address"`
	DownloadPath           string `yaml:"download_path" json:"download_path"`
	MaximumActiveDownloads int    `yaml:"maximum_active_downloads_per_domain" json:"maximum_active_downloads_per_domain"`
}

// DownloadProfile holds the details for executing a downloader
type DownloadProfile struct {
	Name    string   `yaml:"name" json:"name"`
	Command string   `yaml:"command" json:"command"`
	Args    []string `yaml:"args" json:"args"`
}

// DownloadOption contains configuration for extra arguments to pass to the download command
type DownloadOption struct {
	Name string   `yaml:"name" json:"name"`
	Args []string `yaml:"args" json:"args"`
}

// UI holds the configuration for the user interface
type UI struct {
	PopupWidth  int `yaml:"popup_width" json:"popup_width"`
	PopupHeight int `yaml:"popup_height" json:"popup_height"`
}

// Destination is the path for a place that a download can be moved to
type Destination struct {
	Name string `yaml:"name" json:"name"` // Name for this location
	Path string `yaml:"path" json:"path"` // Path on disk
}

// Config is the top level of the user configuration
type Config struct {
	ConfigVersion    int               `yaml:"config_version" json:"config_version"`
	Server           Server            `yaml:"server" json:"server"`
	UI               UI                `yaml:"ui" json:"ui"`
	Destinations     []Destination     `yaml:"destinations" json:"destinations"` // no longer in use, see DownloadOptions
	DownloadProfiles []DownloadProfile `yaml:"profiles" json:"profiles"`
	DownloadOptions  []DownloadOption  `yaml:"download_options" json:"download_options"`
}

// ConfigService is a struct to handle configuration requests, allowing for the
// location that config files are loaded to be customised.
type ConfigService struct {
	Config     *Config
	ConfigPath string
}

func (cs *ConfigService) LoadTestConfig() {
	cs.LoadDefaultConfig()
	cs.Config.Server.DownloadPath = "/tmp"
	cs.Config.DownloadProfiles = []DownloadProfile{{Name: "test profile", Command: "sleep", Args: []string{"5"}}}
}

func (cs *ConfigService) LoadDefaultConfig() {
	defaultConfig := Config{}
	stdProfile := DownloadProfile{Name: "standard video", Command: "yt-dlp", Args: []string{
		"--newline",
		"--write-info-json",
		"-f",
		"bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
	}}
	mp3Profile := DownloadProfile{Name: "standard mp3", Command: "yt-dlp", Args: []string{
		"--newline",
		"--write-info-json",
		"--extract-audio",
		"--audio-format", "mp3",
	}}

	defaultConfig.DownloadProfiles = append(defaultConfig.DownloadProfiles, stdProfile)
	defaultConfig.DownloadProfiles = append(defaultConfig.DownloadProfiles, mp3Profile)

	defaultConfig.Server.Port = 6123
	defaultConfig.Server.Address = "http://localhost:6123"
	defaultConfig.Server.DownloadPath = "/downloads"

	defaultConfig.UI.PopupWidth = 500
	defaultConfig.UI.PopupHeight = 500

	defaultConfig.Server.MaximumActiveDownloads = 2

	defaultConfig.Destinations = nil
	defaultConfig.DownloadOptions = make([]DownloadOption, 0)

	defaultConfig.ConfigVersion = 4

	cs.Config = &defaultConfig

}

// ProfileCalled returns the corresponding DownloadProfile, or nil if it does not exist
func (c *Config) ProfileCalled(name string) *DownloadProfile {
	for _, p := range c.DownloadProfiles {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

// DownloadOptionCalled returns the corresponding DownloadOption, or nil if it does not exist
func (c *Config) DownloadOptionCalled(name string) *DownloadOption {
	for _, o := range c.DownloadOptions {
		if o.Name == name {
			return &o
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
	// check binary path first, for a file called gropple.yml
	binaryPath := os.Args[0]
	binaryDir := filepath.Dir(binaryPath)
	potentialConfigPath := filepath.Join(binaryDir, "gropple.yml")

	_, err := os.Stat(potentialConfigPath)
	if err == nil {
		// exists in binary directory, use that
		// fully qualify, just for clarity in the log
		config, err := filepath.Abs(potentialConfigPath)
		if err == nil {
			log.Printf("found portable config in %s", config)
			cs.ConfigPath = config
			return
		} else {
			log.Printf("got error when trying to convert config to absolute path: %s", err)
			log.Print("falling back to using UserConfigDir")
		}
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
	cs.Config = &c

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

	if c.ConfigVersion == 2 {
		c.Destinations = make([]Destination, 0)
		c.ConfigVersion = 3
		configMigrated = true
		log.Print("migrated config from version 2 => 3")
	}

	if c.ConfigVersion == 3 {
		c.ConfigVersion = 4
		for i := range c.Destinations {
			newDownloadOption := DownloadOption{
				Name: c.Destinations[i].Name,
				Args: []string{"-o", fmt.Sprintf("%s/%%(title)s [%%(id)s].%%(ext)s", c.Destinations[i].Path)},
			}
			c.DownloadOptions = append(c.DownloadOptions, newDownloadOption)
		}
		c.Destinations = nil
		configMigrated = true
		log.Print("migrated config from version 3 => 4")
	}

	if configMigrated {
		log.Print("Writing new config after version migration")
		cs.WriteConfig()
	}

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

	_, err = file.Write(s)
	if err != nil {
		log.Fatalf("could not write config file %s: %s", path, err)
	}
	file.Close()
}
