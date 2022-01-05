package download

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tardisx/gropple/config"
)

type Download struct {
	Id              int                    `json:"id"`
	Url             string                 `json:"url"`
	PopupUrl        string                 `json:"popup_url"`
	Pid             int                    `json:"pid"`
	ExitCode        int                    `json:"exit_code"`
	State           string                 `json:"state"`
	DownloadProfile config.DownloadProfile `json:"download_profile"`
	Finished        bool                   `json:"finished"`
	FinishedTS      time.Time              `json:"finished_ts"`
	Files           []string               `json:"files"`
	Eta             string                 `json:"eta"`
	Percent         float32                `json:"percent"`
	Log             []string               `json:"log"`
	Config          *config.Config
	mutex           sync.Mutex
}

type Downloads []*Download

// StartQueued starts any downloads that have been queued, we would not exceed
// maxRunning. If maxRunning is 0, there is no limit.
func (dls Downloads) StartQueued(maxRunning int) {
	active := make(map[string]int)

	for _, dl := range dls {

		dl.mutex.Lock()

		if dl.State == "downloading" {
			active[dl.domain()]++
		}
		dl.mutex.Unlock()

	}

	for _, dl := range dls {

		dl.mutex.Lock()

		if dl.State == "queued" && (maxRunning == 0 || active[dl.domain()] < maxRunning) {
			dl.State = "downloading"
			active[dl.domain()]++
			log.Printf("Starting download for %#v", dl)
			dl.mutex.Unlock()
			go func() { dl.Begin() }()
		} else {
			dl.mutex.Unlock()
		}
	}

}

// Cleanup removes old downloads from the list. Hardcoded to remove them one hour
// completion.
func (dls Downloads) Cleanup() Downloads {
	newDLs := Downloads{}
	for _, dl := range dls {

		dl.mutex.Lock()

		if dl.Finished && time.Since(dl.FinishedTS) > time.Duration(time.Hour) {
			// do nothing
		} else {
			newDLs = append(newDLs, dl)
		}
		dl.mutex.Unlock()

	}
	return newDLs
}

// Queue queues a download
func (dl *Download) Queue() {

	dl.mutex.Lock()
	defer dl.mutex.Unlock()

	dl.State = "queued"

}

func (dl *Download) Stop() {
	log.Printf("stopping the download")
	dl.mutex.Lock()
	defer dl.mutex.Unlock()

	syscall.Kill(dl.Pid, syscall.SIGTERM)
}

func (dl *Download) domain() string {

	// note that we expect to already have the mutex locked by the caller
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

	dl.mutex.Lock()

	dl.State = "downloading"
	cmdSlice := []string{}
	cmdSlice = append(cmdSlice, dl.DownloadProfile.Args...)

	// only add the url if it's not empty. This helps us with testing
	if dl.Url != "" {
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
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		dl.State = "failed"
		dl.Finished = true
		dl.FinishedTS = time.Now()
		dl.Log = append(dl.Log, fmt.Sprintf("error setting up stderr pipe: %v", err))
		return
	}

	log.Printf("Starting %v", cmd)
	err = cmd.Start()
	if err != nil {
		dl.State = "failed"
		dl.Finished = true
		dl.FinishedTS = time.Now()
		dl.Log = append(dl.Log, fmt.Sprintf("error starting command '%s': %v", dl.DownloadProfile.Command, err))
		return
	}
	dl.Pid = cmd.Process.Pid

	var wg sync.WaitGroup

	dl.mutex.Unlock()

	wg.Add(2)
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

	dl.mutex.Lock()

	dl.State = "complete"
	dl.Finished = true
	dl.FinishedTS = time.Now()
	dl.ExitCode = cmd.ProcessState.ExitCode()

	if dl.ExitCode != 0 {
		dl.State = "failed"
	}
	dl.mutex.Unlock()

}

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

				dl.mutex.Lock()

				// append the raw log
				dl.Log = append(dl.Log, l)

				dl.mutex.Unlock()

				// look for the percent and eta and other metadata
				dl.updateMetadata(l)
			}
		}
		if err != nil {
			break
		}
	}
}

func (dl *Download) updateMetadata(s string) {

	dl.mutex.Lock()

	defer dl.mutex.Unlock()

	// [download]  49.7% of ~15.72MiB at  5.83MiB/s ETA 00:07
	etaRE := regexp.MustCompile(`download.+ETA +(\d\d:\d\d(?::\d\d)?)$`)
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
}
