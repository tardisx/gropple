package download

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tardisx/gropple/config"
)

type Download struct {
	Id              int                    `json:"id"`
	Url             string                 `json:"url"`
	PopupUrl        string                 `json:"popup_url"`
	Process         *os.Process            `json:"-"`
	ExitCode        int                    `json:"exit_code"`
	State           string                 `json:"state"`
	DownloadProfile config.DownloadProfile `json:"download_profile"`
	Destination     *config.Destination    `json:"destination"`
	Finished        bool                   `json:"finished"`
	FinishedTS      time.Time              `json:"finished_ts"`
	Files           []string               `json:"files"`
	PlaylistCurrent int                    `json:"playlist_current"`
	PlaylistTotal   int                    `json:"playlist_total"`
	Eta             string                 `json:"eta"`
	Percent         float32                `json:"percent"`
	Log             []string               `json:"log"`
	Config          *config.Config
	Lock            sync.Mutex
}

// The Manager holds and is responsible for all Download objects.
type Manager struct {
	Downloads    []*Download
	MaxPerDomain int
	Lock         sync.Mutex
}

var CanStopDownload = false

var downloadId int32 = 0

func (m *Manager) ManageQueue() {
	for {
		m.Lock.Lock()

		m.startQueued(m.MaxPerDomain)
		// m.cleanup()
		m.Lock.Unlock()

		time.Sleep(time.Second)
	}
}

// startQueued starts any downloads that have been queued, we would not exceed
// maxRunning. If maxRunning is 0, there is no limit.
func (m *Manager) startQueued(maxRunning int) {

	active := make(map[string]int)

	for _, dl := range m.Downloads {
		dl.Lock.Lock()

		if dl.State == "downloading" || dl.State == "preparing to start" {
			active[dl.domain()]++
		}
		dl.Lock.Unlock()

	}

	for _, dl := range m.Downloads {

		dl.Lock.Lock()

		if dl.State == "queued" && (maxRunning == 0 || active[dl.domain()] < maxRunning) {
			dl.State = "preparing to start"
			active[dl.domain()]++
			log.Printf("Starting download for id:%d (%s)", dl.Id, dl.Url)

			dl.Lock.Unlock()

			go func(sdl *Download) {
				sdl.Begin()
			}(dl)
		} else {
			dl.Lock.Unlock()
		}

	}

}

// cleanup removes old downloads from the list. Hardcoded to remove them one hour
// completion.
func (m *Manager) XXXcleanup() {
	newDLs := []*Download{}
	for _, dl := range m.Downloads {

		if dl.Finished && time.Since(dl.FinishedTS) > time.Duration(time.Hour) {
			// do nothing
		} else {
			newDLs = append(newDLs, dl)
		}

	}
	m.Downloads = newDLs
}

// GetDlById returns one of the downloads in our current list.
func (m *Manager) GetDlById(id int) (*Download, error) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	for _, dl := range m.Downloads {
		if dl.Id == id {
			return dl, nil
		}
	}
	return nil, fmt.Errorf("no download with id %d", id)
}

// Queue queues a download
func (m *Manager) Queue(dl *Download) {
	dl.Lock.Lock()
	defer dl.Lock.Unlock()
	dl.State = "queued"
}

func NewDownload(url string, conf *config.Config) *Download {
	atomic.AddInt32(&downloadId, 1)
	dl := Download{
		Id:       int(downloadId),
		Url:      url,
		PopupUrl: fmt.Sprintf("/fetch/%d", int(downloadId)),
		State:    "choose profile",
		Files:    []string{},
		Log:      []string{},
		Config:   conf,
		Lock:     sync.Mutex{},
	}
	return &dl
}

func (m *Manager) AddDownload(dl *Download) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.Downloads = append(m.Downloads, dl)
}

// func (dl *Download) AppendLog(text string) {
// 	dl.Lock.Lock()
// 	defer dl.Lock.Unlock()
// 	dl.Log = append(dl.Log, text)
// }

// Stop the download.
func (dl *Download) Stop() {
	if !CanStopDownload {
		log.Print("attempted to stop download on a platform that it is not currently supported on - please report this as a bug")
		os.Exit(1)
	}

	log.Printf("stopping the download")
	dl.Lock.Lock()
	defer dl.Lock.Unlock()
	dl.Log = append(dl.Log, "aborted by user")
	dl.Process.Kill()
}

// domain returns a domain for this Download. Download should be locked.
func (dl *Download) domain() string {

	url, err := url.Parse(dl.Url)
	if err != nil {
		log.Printf("Unknown domain for url: %s", dl.Url)
		return "unknown"
	}

	return url.Hostname()

}

// Begin starts a download, by starting the command specified in the DownloadProfile.
// It blocks until the download is complete.
func (dl *Download) Begin() {
	dl.Lock.Lock()

	dl.State = "downloading"
	cmdSlice := []string{}
	cmdSlice = append(cmdSlice, dl.DownloadProfile.Args...)

	// only add the url if it's not empty or an example URL. This helps us with testing
	if !(dl.Url == "" || strings.Contains(dl.domain(), "example.org")) {
		cmdSlice = append(cmdSlice, dl.Url)
	}

	cmd := exec.Command(dl.DownloadProfile.Command, cmdSlice...)
	cmd.Dir = dl.Config.Server.DownloadPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		dl.State = "failed"
		dl.Finished = true
		dl.FinishedTS = time.Now()
		dl.Log = append(dl.Log, fmt.Sprintf("error setting up stdout pipe: %v", err))
		dl.Lock.Unlock()

		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		dl.State = "failed"
		dl.Finished = true
		dl.FinishedTS = time.Now()
		dl.Log = append(dl.Log, fmt.Sprintf("error setting up stderr pipe: %v", err))
		dl.Lock.Unlock()

		return
	}

	log.Printf("Executing command: %v", cmd)
	err = cmd.Start()
	if err != nil {
		dl.State = "failed"
		dl.Finished = true
		dl.FinishedTS = time.Now()
		dl.Log = append(dl.Log, fmt.Sprintf("error starting command '%s': %v", dl.DownloadProfile.Command, err))
		dl.Lock.Unlock()

		return
	}
	dl.Process = cmd.Process

	var wg sync.WaitGroup

	wg.Add(2)

	dl.Lock.Unlock()

	go func() {
		defer wg.Done()
		dl.updateDownload(stdout)
	}()

	go func() {
		defer wg.Done()
		dl.updateDownload(stderr)
	}()

	wg.Wait()
	cmd.Wait()

	dl.Lock.Lock()

	log.Printf("Process finished for id: %d (%v)", dl.Id, cmd)

	dl.State = "complete"
	dl.Finished = true
	dl.FinishedTS = time.Now()
	dl.ExitCode = cmd.ProcessState.ExitCode()

	if dl.ExitCode != 0 {
		dl.State = "failed"
	}
	dl.Lock.Unlock()

}

// updateDownload updates the download based on data from the reader. Expects the
// Download to be unlocked.
func (dl *Download) updateDownload(r io.Reader) {

	// XXX not sure if we might get a partial line?
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			s := string(buf[:n])
			lines := strings.Split(s, "\n")

			for _, l := range lines {

				if l == "" {
					continue
				}

				// append the raw log
				dl.Lock.Lock()
				dl.Log = append(dl.Log, l)
				// look for the percent and eta and other metadata
				dl.updateMetadata(l)
				dl.Lock.Unlock()

			}
		}
		if err != nil {
			break
		}
	}
}

// updateMetadata parses some metadata and updates the Download. Download must be locked.
func (dl *Download) updateMetadata(s string) {

	// [download]  49.7% of ~15.72MiB at  5.83MiB/s ETA 00:07
	// [download]  99.3% of ~1.42GiB at 320.87KiB/s ETA 00:07 (frag 212/214)
	etaRE := regexp.MustCompile(`download.+ETA +(\d\d:\d\d(?::\d\d)?)`)
	matches := etaRE.FindStringSubmatch(s)
	if len(matches) == 2 {
		dl.Eta = matches[1]
		dl.State = "downloading"

	}

	percentRE := regexp.MustCompile(`download.+?([\d\.]+)%`)
	matches = percentRE.FindStringSubmatch(s)
	if len(matches) == 2 {
		p, err := strconv.ParseFloat(matches[1], 32)
		if err == nil {
			dl.Percent = float32(p)
		} else {
			panic(err)
		}
	}

	// This appears once per destination file
	// [download] Destination: Filename with spaces and other punctuation here be careful!.mp4
	filename := regexp.MustCompile(`download.+?Destination: (.+)$`)
	matches = filename.FindStringSubmatch(s)
	if len(matches) == 2 {
		dl.Files = append(dl.Files, matches[1])
	}

	// This means a file has been "created" by merging others
	// [ffmpeg] Merging formats into "Toto - Africa (Official HD Video)-FTQbiNvZqaY.mp4"
	mergedFilename := regexp.MustCompile(`Merging formats into "(.+)"$`)
	matches = mergedFilename.FindStringSubmatch(s)
	if len(matches) == 2 {
		dl.Files = append(dl.Files, matches[1])
	}

	// This means a file has been deleted
	// Gross - this time it's unquoted and has trailing guff
	// Deleting original file Toto - Africa (Official HD Video)-FTQbiNvZqaY.f137.mp4 (pass -k to keep)
	// This is very fragile
	deletedFile := regexp.MustCompile(`Deleting original file (.+) \(pass -k to keep\)$`)
	matches = deletedFile.FindStringSubmatch(s)
	if len(matches) == 2 {
		// find the index
		for i, f := range dl.Files {
			if f == matches[1] {
				dl.Files = append(dl.Files[:i], dl.Files[i+1:]...)
				break
			}
		}
	}

	// [download] Downloading video 1 of 3
	playlistDetails := regexp.MustCompile(`Downloading video (\d+) of (\d+)`)
	matches = playlistDetails.FindStringSubmatch(s)
	if len(matches) == 3 {
		total, _ := strconv.ParseInt(matches[2], 10, 32)
		current, _ := strconv.ParseInt(matches[1], 10, 32)
		dl.PlaylistTotal = int(total)
		dl.PlaylistCurrent = int(current)
	}

	// [Site] user: Downloading JSON metadata page 2
	metadataDL := regexp.MustCompile(`Downloading JSON metadata page (\d+)`)
	matches = metadataDL.FindStringSubmatch(s)
	if len(matches) == 2 {
		dl.State = "Downloading metadata, page " + matches[1]
	}

	// [FixupM3u8] Fixing MPEG-TS in MP4 container of "file [-168849776_456239489].mp4"
	metadataFixup := regexp.MustCompile(`Fixing MPEG-TS in MP4 container`)
	matches = metadataFixup.FindStringSubmatch(s)
	if len(matches) == 1 {
		dl.State = "Fixing MPEG-TS in MP4"
	}

}
