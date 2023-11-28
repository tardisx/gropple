package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrationV1toV4(t *testing.T) {
	v1Config := `config_version: 1
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
	cs := configServiceFromString(v1Config)
	err := cs.LoadConfig()
	if err != nil {
		t.Errorf("got error when loading config: %s", err)
	}
	if cs.Config.ConfigVersion != 4 {
		t.Errorf("did not migrate version (it is '%d')", cs.Config.ConfigVersion)
	}
	if cs.Config.Server.MaximumActiveDownloads != 2 {
		t.Error("did not add MaximumActiveDownloads")
	}
	if len(cs.Config.Destinations) != 0 {
		t.Error("incorrect number of destinations added")
	}
	os.Remove(cs.ConfigPath)
}

func TestMigrateV3toV4(t *testing.T) {
	v3Config := `config_version: 3
server:
  port: 6123
  address: http://localhost:6123
  download_path: /tmp/Downloads
  maximum_active_downloads_per_domain: 2
ui:
  popup_width: 900
  popup_height: 900
destinations:
  - name: cool destination
    path: /tmp/coolness
profiles:
  - name: standard video
    command: yt-dlp
    args:
      - --newline
      - --write-info-json
      - -f
      - bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best
  - name: standard mp3
    command: yt-dlp
    args:
      - --newline
      - --write-info-json
      - --extract-audio
      - --audio-format
      - mp3`
	cs := configServiceFromString(v3Config)
	err := cs.LoadConfig()
	if err != nil {
		t.Errorf("got error when loading config: %s", err)
	}
	if cs.Config.ConfigVersion != 4 {
		t.Errorf("did not migrate version (it is '%d')", cs.Config.ConfigVersion)
	}
	if cs.Config.Server.MaximumActiveDownloads != 2 {
		t.Error("did not add MaximumActiveDownloads")
	}
	if len(cs.Config.Destinations) != 0 {
		t.Error("incorrect number of destinations from migrated file")
	}
	if assert.Len(t, cs.Config.DownloadOptions, 1) {
		if assert.Len(t, cs.Config.DownloadOptions[0].Args, 2) {
			assert.Equal(t, "-o", cs.Config.DownloadOptions[0].Args[0])
			assert.Equal(t, "/tmp/coolness/%(title)s [%(id)s].%(ext)s", cs.Config.DownloadOptions[0].Args[1])
		}
	}
	os.Remove(cs.ConfigPath)
}

func TestMigrateV3toV4CrashBug(t *testing.T) {
	v3Config := `config_version: 3
server:
  port: 6123
  address: https://superaddress.here.com
  download_path: /home/path/gropple
  maximum_active_downloads_per_domain: 2
ui:
  popup_width: 500
  popup_height: 500
destinations:
- name: somegifs
  path: /home/path/somegifs
- name: otherstuff
  path: /home/path/otherstuff
profiles:
- name: standard video
  command: yt-dlp
  args:
  - --newline
  - --write-info-json
  - -f
  - bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best
  - --verbose
  - --embed-metadata
  - --embed-subs
  - --embed-thumbnail
- name: standard mp3
  command: yt-dlp
  args:
  - --extract-audio
  - --audio-format
  - mp3
  - --prefer-ffmpeg
`
	cs := configServiceFromString(v3Config)
	err := cs.LoadConfig()
	if err != nil {
		t.Errorf("got error when loading config: %s", err)
	}
	if cs.Config.ConfigVersion != 4 {
		t.Errorf("did not migrate version (it is '%d')", cs.Config.ConfigVersion)
	}
	if cs.Config.Server.MaximumActiveDownloads != 2 {
		t.Error("did not add MaximumActiveDownloads")
	}
	if len(cs.Config.Destinations) != 0 {
		t.Error("incorrect number of destinations from migrated file")
	}
	if assert.Len(t, cs.Config.DownloadOptions, 2) {
		if assert.Len(t, cs.Config.DownloadOptions[0].Args, 2) {
			assert.Equal(t, "-o", cs.Config.DownloadOptions[0].Args[0])
			assert.Equal(t, "/home/path/somegifs/%(title)s [%(id)s].%(ext)s", cs.Config.DownloadOptions[0].Args[1])
			assert.Equal(t, "-o", cs.Config.DownloadOptions[1].Args[0])
			assert.Equal(t, "/home/path/otherstuff/%(title)s [%(id)s].%(ext)s", cs.Config.DownloadOptions[1].Args[1])
		}
	}
	os.Remove(cs.ConfigPath)
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

func TestLookForExecutable(t *testing.T) {
	cmdPath, err := exec.LookPath("sleep")
	if err != nil {
		t.Errorf("cannot run this test without knowing about sleep: %s", err)
		t.FailNow()
	}
	cmdDir := filepath.Dir(cmdPath)

	cmd := "sleep"
	path, err := AbsPathToExecutable(cmd)
	if assert.NoError(t, err) {
		assert.Equal(t, cmdPath, path)
	}

	cmd = cmdPath
	path, err = AbsPathToExecutable(cmd)
	if assert.NoError(t, err) {
		assert.Equal(t, cmdPath, path)
	}

	cmd = "../../../../../../../../.." + cmdPath
	path, err = AbsPathToExecutable(cmd)
	if assert.NoError(t, err) {
		assert.Equal(t, cmdPath, path)
	}
	cmd = "./sleep"
	_, err = AbsPathToExecutable(cmd)
	assert.Error(t, err)

	os.Chdir(cmdDir)
	cmd = "./sleep"
	path, err = AbsPathToExecutable(cmd)
	if assert.NoError(t, err) {
		assert.Equal(t, cmdPath, path)
	}

}
