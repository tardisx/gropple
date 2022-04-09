package config

import (
	"os"
	"testing"
)

func TestMigrationV1toV2(t *testing.T) {
	v2Config := `config_version: 1
server:
  port: 6123
  address: http://localhost:6123
  download_path: ./
ui:
  popup_width: 500
  popup_height: 500
profiles:
- name: standard video
  command: youtube-dl
  args:
  - --newline
  - --write-info-json
  - -f
  - bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best
- name: standard mp3
  command: youtube-dl
  args:
  - --newline
  - --write-info-json
  - --extract-audio
  - --audio-format
  - mp3
`
	cs := configServiceFromString(v2Config)
	err := cs.LoadConfig()
	if err != nil {
		t.Errorf("got error when loading config: %s", err)
	}
	if cs.Config.ConfigVersion != 2 {
		t.Errorf("did not migrate version (it is '%d')", cs.Config.ConfigVersion)
	}
	if cs.Config.Server.MaximumActiveDownloads != 2 {
		t.Error("did not add MaximumActiveDownloads")
	}
	t.Log(cs.ConfigPath)
}

func configServiceFromString(configString string) *ConfigService {
	tmpFile, _ := os.CreateTemp("", "gropple_test_*.yml")
	tmpFile.Write([]byte(configString))
	tmpFile.Close()
	cs := ConfigService{
		Config:     &Config{},
		ConfigPath: tmpFile.Name(),
	}
	return &cs
}
