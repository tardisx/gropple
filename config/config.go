package config

import (
	"log"

	"gopkg.in/yaml.v2"
)

type Server struct {
	Port         int    `yaml:"port"`
	Address      string `yaml:"address"`
	DownloadPath string `yaml:"download_path"`
}

type DownloadProfile struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

type UI struct {
	PopupWidth  int `yaml:"popup_width"`
	PopupHeight int `yaml:"popup_height"`
}

type Config struct {
	Server           Server            `yaml:"server"`
	UI               UI                `yaml:"ui"`
	DownloadProfiles []DownloadProfile `yaml:"profiles"`
	ConfigVersion    int               `yaml:"config_version"`
}

func DefaultConfig() Config {
	defaultConfig := Config{}
	stdProfile := DownloadProfile{Name: "standard youtube-dl video", Command: "youtube-dl", Args: []string{
		"--newline",
		"--write-info-json",
		"-f",
		"bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
	}}

	defaultConfig.DownloadProfiles = append(defaultConfig.DownloadProfiles, stdProfile)
	defaultConfig.Server.Port = 6123
	defaultConfig.Server.Address = "localhost:6123"
	defaultConfig.Server.DownloadPath = "./"
	defaultConfig.UI.PopupWidth = 500
	defaultConfig.UI.PopupHeight = 500

	return defaultConfig
}

func WriteDefaultConfig(path string) {
	defaultConfig := DefaultConfig()
	s, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		panic(err)
	}
	log.Print(string(s))
}
