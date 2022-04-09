package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"strconv"

	"github.com/gorilla/mux"
	"github.com/tardisx/gropple/config"
	"github.com/tardisx/gropple/download"
	"github.com/tardisx/gropple/version"
)

var downloads download.Downloads
var downloadId = 0
var configService *config.ConfigService

var versionInfo = version.Info{CurrentVersion: "v0.6.0"}

//go:embed web
var webFS embed.FS

type successResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type errorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func main() {
	log.Printf("Starting gropple %s - https://github.com/tardisx/gropple", versionInfo.CurrentVersion)

	configService = &config.ConfigService{}
	configService.DetermineConfigDir()
	exists, err := configService.ConfigFileExists()
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		log.Print("No config file - creating default config")
		configService.LoadDefaultConfig()
		configService.WriteConfig()
		log.Printf("Configuration written to %s", configService.ConfigPath)
	} else {
		err := configService.LoadConfig()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Configuration loaded from %s", configService.ConfigPath)

	}

	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/static/{filename}", staticHandler)
	r.HandleFunc("/config", configHandler)
	r.HandleFunc("/fetch", fetchHandler)
	r.HandleFunc("/fetch/{id}", fetchHandler)

	// info for the list
	r.HandleFunc("/rest/fetch", fetchInfoRESTHandler)
	// info for one, including update
	r.HandleFunc("/rest/fetch/{id}", fetchInfoOneRESTHandler)
	r.HandleFunc("/rest/version", versionRESTHandler)
	r.HandleFunc("/rest/config", configRESTHandler)

	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf(":%d", configService.Config.Server.Port),
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

	// start downloading queued downloads when slots available, and clean up
	// old entries
	go func() {
		for {
			downloads.StartQueued(configService.Config.Server.MaximumActiveDownloads)
			downloads = downloads.Cleanup()
			time.Sleep(time.Second)
		}
	}()

	log.Printf("Visit %s for details on installing the bookmarklet and to check status", configService.Config.Server.Address)
	log.Fatal(srv.ListenAndServe())

}

// versionRESTHandler returns the version information, if we have up-to-date info from github
func versionRESTHandler(w http.ResponseWriter, r *http.Request) {
	if versionInfo.GithubVersionFetched {
		b, _ := json.Marshal(versionInfo)
		w.Write(b)
	} else {
		w.WriteHeader(400)
	}
}

// homeHandler returns the main index page
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	bookmarkletURL := fmt.Sprintf("javascript:(function(f,s,n,o){window.open(f+encodeURIComponent(s),n,o)}('%s/fetch?url=',window.location,'yourform','width=%d,height=%d'));", configService.Config.Server.Address, configService.Config.UI.PopupWidth, configService.Config.UI.PopupHeight)

	t, err := template.ParseFS(webFS, "web/layout.tmpl", "web/menu.tmpl", "web/index.html")
	if err != nil {
		panic(err)
	}

	type Info struct {
		Downloads      []*download.Download
		BookmarkletURL template.URL
		Config         *config.Config
		Version        version.Info
	}

	info := Info{
		Downloads:      downloads,
		BookmarkletURL: template.URL(bookmarkletURL),
		Config:         configService.Config,
		Version:        versionInfo,
	}

	err = t.ExecuteTemplate(w, "layout", info)
	if err != nil {
		panic(err)
	}
}

// staticHandler handles requests for static files
func staticHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	filename := vars["filename"]
	if strings.Index(filename, ".js") == len(filename)-3 {
		f, err := webFS.Open("web/" + filename)
		if err != nil {
			log.Printf("error accessing %s - %v", filename, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		io.Copy(w, f)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// configHandler returns the configuration page
func configHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	t, err := template.ParseFS(webFS, "web/layout.tmpl", "web/menu.tmpl", "web/config.html")
	if err != nil {
		panic(err)
	}

	err = t.ExecuteTemplate(w, "layout", nil)
	if err != nil {
		panic(err)
	}
}

// configRESTHandler handles both reading and writing of the configuration
func configRESTHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		log.Printf("Updating config")
		b, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		err = configService.Config.UpdateFromJSON(b)

		if err != nil {
			errorRes := errorResponse{Success: false, Error: err.Error()}
			errorResB, _ := json.Marshal(errorRes)
			w.WriteHeader(400)
			w.Write(errorResB)
			return
		}
		configService.WriteConfig()
	}
	b, _ := json.Marshal(configService.Config)
	w.Write(b)
}

//
func fetchInfoOneRESTHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idString := vars["id"]
	if idString != "" {
		id, err := strconv.Atoi(idString)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// find the download
		var thisDownload *download.Download
		for _, dl := range downloads {
			if dl.Id == id {
				thisDownload = dl
			}
		}
		if thisDownload == nil {
			http.NotFound(w, r)
			return
		}

		if r.Method == "POST" {

			type updateRequest struct {
				Action  string `json:"action"`
				Profile string `json:"profile"`
			}

			thisReq := updateRequest{}

			b, err := io.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}

			err = json.Unmarshal(b, &thisReq)
			if err != nil {
				errorRes := errorResponse{Success: false, Error: err.Error()}
				errorResB, _ := json.Marshal(errorRes)
				w.WriteHeader(400)
				w.Write(errorResB)
				return
			}

			if thisReq.Action == "start" {
				// find the profile they asked for
				profile := configService.Config.ProfileCalled(thisReq.Profile)
				if profile == nil {
					panic("bad profile name?")
				}
				// set the profile
				thisDownload.DownloadProfile = *profile

				thisDownload.Queue()
				succRes := successResponse{Success: true, Message: "download started"}
				succResB, _ := json.Marshal(succRes)
				w.Write(succResB)
				return
			}

			if thisReq.Action == "stop" {
				thisDownload.Stop()
				succRes := successResponse{Success: true, Message: "download stopped"}
				succResB, _ := json.Marshal(succRes)
				w.Write(succResB)
				return
			}
		}

		// just a get, return the object
		b, _ := json.Marshal(thisDownload)
		w.Write(b)
		return
	} else {
		http.NotFound(w, r)
	}
}

func fetchInfoRESTHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := json.Marshal(downloads)
	w.Write(b)
}

func fetchHandler(w http.ResponseWriter, r *http.Request) {

	// if they refreshed the popup, just load the existing object, don't
	// create a new one
	vars := mux.Vars(r)
	idString := vars["id"]

	idInt, err := strconv.ParseInt(idString, 10, 32)
	if err == nil && idInt > 0 {
		for _, dl := range downloads {
			if dl.Id == int(idInt) {
				t, err := template.ParseFS(webFS, "web/layout.tmpl", "web/popup.html")
				if err != nil {
					panic(err)
				}

				templateData := map[string]interface{}{"dl": dl, "config": configService.Config, "canStop": download.CanStopDownload}

				err = t.ExecuteTemplate(w, "layout", templateData)
				if err != nil {
					panic(err)
				}
				return
			}
		}

	}

	query := r.URL.Query()
	url, present := query["url"]

	if !present {
		w.WriteHeader(400)
		fmt.Fprint(w, "No url supplied")
		return
	} else {

		// check the URL for a sudden but inevitable betrayal
		if strings.Contains(url[0], configService.Config.Server.Address) {
			w.WriteHeader(400)
			fmt.Fprint(w, "you mustn't gropple your gropple :-)")
			return
		}

		// create the record
		newDownload := download.NewDownload(configService.Config, url[0])
		downloads = append(downloads, newDownload)
		// XXX atomic ^^

		newDownload.Log = append(newDownload.Log, "start of log...")

		// go func() {
		// 	newDownload.Begin()
		// }()

		t, err := template.ParseFS(webFS, "web/layout.tmpl", "web/popup.html")
		if err != nil {
			panic(err)
		}

		templateData := map[string]interface{}{"Version": versionInfo, "dl": newDownload, "config": configService.Config, "canStop": download.CanStopDownload}

		err = t.ExecuteTemplate(w, "layout", templateData)
		if err != nil {
			panic(err)
		}
	}
}
