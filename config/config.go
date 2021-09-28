package config

import (
	"log"

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
	Server           Server            `yaml:"server" json:"server"`
	UI               UI                `yaml:"ui" json:"ui"`
	DownloadProfiles []DownloadProfile `yaml:"profiles" json:"profiles"`
	ConfigVersion    int               `yaml:"config_version" json:"config_version"`
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
	defaultConfig.Server.Address = "http://localhost:6123"
	defaultConfig.Server.DownloadPath = "./"

	defaultConfig.UI.PopupWidth = 500
	defaultConfig.UI.PopupHeight = 500

	defaultConfig.ConfigVersion = 1

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
