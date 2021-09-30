package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"strconv"

	"github.com/gorilla/mux"
	"github.com/tardisx/gropple/config"
	"github.com/tardisx/gropple/version"
)

type download struct {
	Id       int      `json:"id"`
	Url      string   `json:"url"`
	Pid      int      `json:"pid"`
	ExitCode int      `json:"exit_code"`
	State    string   `json:"state"`
	Finished bool     `json:"finished"`
	Files    []string `json:"files"`
	Eta      string   `json:"eta"`
	Percent  float32  `json:"percent"`
	Log      []string `json:"log"`
}

var downloads []*download
var downloadId = 0

var versionInfo = version.Info{CurrentVersion: "v0.5.0"}

//go:embed web
var webFS embed.FS

var conf *config.Config

func main() {
	if !config.ConfigFileExists() {
		log.Print("No config file - creating default config")
		conf = config.DefaultConfig()
		conf.WriteConfig()
	} else {
		loadedConfig, err := config.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}
		conf = loadedConfig
	}

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/config", ConfigHandler)
	r.HandleFunc("/fetch", FetchHandler)

	r.HandleFunc("/rest/fetch/info", FetchInfoHandler)
	r.HandleFunc("/rest/fetch/info/{id}", FetchInfoOneHandler)
	r.HandleFunc("/rest/version", VersionHandler)
	r.HandleFunc("/rest/config", ConfigRESTHandler)

	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf(":%d", conf.Server.Port),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}

	// check for a new version every 4 hours
	go func() {
		for {
			versionInfo.UpdateGitHubVersion()
			time.Sleep(time.Hour * 4)
		}
	}()

	log.Printf("starting gropple %s - https://github.com/tardisx/gropple", versionInfo.CurrentVersion)
	log.Printf("go to %s for details on installing the bookmarklet and to check status", conf.Server.Address)
	log.Fatal(srv.ListenAndServe())
}

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	if versionInfo.GithubVersionFetched {
		b, _ := json.Marshal(versionInfo)
		w.Write(b)
	} else {
		w.WriteHeader(400)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	bookmarkletURL := fmt.Sprintf("javascript:(function(f,s,n,o){window.open(f+encodeURIComponent(s),n,o)}('%s/fetch?url=',window.location,'yourform','width=%d,height=%d'));", conf.Server.Address, conf.UI.PopupWidth, conf.UI.PopupHeight)

	t, err := template.ParseFS(webFS, "web/layout.tmpl", "web/index.html")
	if err != nil {
		panic(err)
	}

	type Info struct {
		Downloads      []*download
		BookmarkletURL template.URL
	}

	info := Info{
		Downloads:      downloads,
		BookmarkletURL: template.URL(bookmarkletURL),
	}

	err = t.ExecuteTemplate(w, "layout", info)
	if err != nil {
		panic(err)
	}

}

func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	t, err := template.ParseFS(webFS, "web/layout.tmpl", "web/config.html")
	if err != nil {
		panic(err)
	}

	err = t.ExecuteTemplate(w, "layout", nil)
	if err != nil {
		panic(err)
	}
}

func ConfigRESTHandler(w http.ResponseWriter, r *http.Request) {

	type errorResponse struct {
		Error string `json:"error"`
	}

	if r.Method == "POST" {
		log.Printf("Updating config")
		b, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		err = conf.UpdateFromJSON(b)

		if err != nil {
			errorRes := errorResponse{Error: err.Error()}
			errorResB, _ := json.Marshal(errorRes)
			w.WriteHeader(400)
			w.Write(errorResB)
			return
		}
		conf.WriteConfig()
	}
	b, _ := json.Marshal(conf)
	w.Write(b)
}

func FetchInfoOneHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idString := vars["id"]
	if idString != "" {
		id, err := strconv.Atoi(idString)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		for _, dl := range downloads {
			if dl.Id == id {
				b, _ := json.Marshal(dl)
				w.Write(b)
				return
			}
		}
	} else {
		http.NotFound(w, r)
	}
}

func FetchInfoHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := json.Marshal(downloads)
	w.Write(b)
}

func FetchHandler(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()
	url, present := query["url"]

	if !present {
		w.WriteHeader(400)
		fmt.Fprint(w, "No url supplied")
		return
	} else {

		// check the URL for a sudden but inevitable betrayal
		if strings.Contains(url[0], conf.Server.Address) {
			w.WriteHeader(400)
			fmt.Fprint(w, "you musn't gropple your gropple :-)")
			return
		}

		// create the record
		// XXX should be atomic!
		downloadId++
		newDownload := download{
			Id:       downloadId,
			Url:      url[0],
			State:    "starting",
			Finished: false,
			Eta:      "?",
			Percent:  0.0,
			Log:      make([]string, 0, 1000),
		}
		downloads = append(downloads, &newDownload)
		// XXX atomic ^^

		newDownload.Log = append(newDownload.Log, "start of log...")

		go func() {
			queue(&newDownload)
		}()

		t, err := template.ParseFS(webFS, "web/layout.tmpl", "web/popup.html")
		if err != nil {
			panic(err)
		}
		err = t.ExecuteTemplate(w, "layout", newDownload)
		if err != nil {
			panic(err)
		}
	}
}

func queue(dl *download) {
	cmdSlice := []string{}
	cmdSlice = append(cmdSlice, conf.DownloadProfiles[0].Args...)
	cmdSlice = append(cmdSlice, dl.Url)

	cmd := exec.Command(conf.DownloadProfiles[0].Command, cmdSlice...)
	cmd.Dir = conf.Server.DownloadPath

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		dl.State = "failed"
		dl.Finished = true
		dl.Log = append(dl.Log, fmt.Sprintf("error setting up stdout pipe: %v", err))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		dl.State = "failed"
		dl.Finished = true
		dl.Log = append(dl.Log, fmt.Sprintf("error setting up stderr pipe: %v", err))
		return
	}

	err = cmd.Start()
	if err != nil {
		dl.State = "failed"
		dl.Finished = true
		dl.Log = append(dl.Log, fmt.Sprintf("error starting youtube-dl: %v", err))
		return
	}
	dl.Pid = cmd.Process.Pid

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		updateDownload(stdout, dl)
	}()

	go func() {
		defer wg.Done()
		updateDownload(stderr, dl)
	}()

	wg.Wait()
	cmd.Wait()

	dl.State = "complete"
	dl.Finished = true
	dl.ExitCode = cmd.ProcessState.ExitCode()

	if dl.ExitCode != 0 {
		dl.State = "failed"
	}

}

func updateDownload(r io.Reader, dl *download) {
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
				dl.Log = append(dl.Log, l)

				// look for the percent and eta and other metadata
				updateMetadata(dl, l)
			}
		}
		if err != nil {
			break
		}
	}
}

func updateMetadata(dl *download, s string) {

	// [download]  49.7% of ~15.72MiB at  5.83MiB/s ETA 00:07
	etaRE := regexp.MustCompile(`download.+ETA +(\d\d:\d\d)`)
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
