package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type Server struct {
	Port         int    `yaml:"port" json:"port"`
	Address      string `yaml:"address" json:"address"`
	DownloadPath string `yaml:"download_path" json:"download_path"`
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

func DefaultConfig() *Config {
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

	defaultConfig.ConfigVersion = 1

	return &defaultConfig
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

	}

	*c = newConfig
	return nil
}

func configPath() string {
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
	return fullFilename
}

func ConfigFileExists() bool {
	info, err := os.Stat(configPath())
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		log.Fatal(err)
	}
	if info.Size() == 0 {
		log.Print("config file is 0 bytes?")
		return false
	}
	return true
}

func LoadConfig() (*Config, error) {
	path := configPath()
	b, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Could not read config '%s': %v", path, err)
		return nil, err
	}
	c := Config{}
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		log.Printf("Could not parse YAML config '%s': %v", path, err)
		return nil, err
	}
	return &c, nil
}

func (c *Config) WriteConfig() {
	s, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}

	path := configPath()
	file, err := os.Create(
		path,
	)

	if err != nil {
		log.Fatalf("Could not open config file")
	}
	defer file.Close()

	file.Write(s)
	file.Close()

	log.Printf("Wrote configuration out to %s", path)
}
