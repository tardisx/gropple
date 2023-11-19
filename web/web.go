package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tardisx/gropple/config"
	"github.com/tardisx/gropple/download"
	"github.com/tardisx/gropple/version"
)

type successResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type queuedResponse struct {
	Success  bool   `json:"success"`
	Location string `json:"location"`
}

type errorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

//go:embed data/**
var webFS embed.FS

func CreateRoutes(cs *config.ConfigService, dm *download.Manager, vm *version.Manager) *mux.Router {
	r := mux.NewRouter()

	// main index page
	r.HandleFunc("/", homeHandler(cs, vm, dm))
	// update info on the status page
	r.HandleFunc("/rest/fetch", fetchInfoRESTHandler(dm))

	// return static files
	r.HandleFunc("/static/{filename}", staticHandler())

	// return the config page
	r.HandleFunc("/config", configHandler())
	// handle config fetches/updates
	r.HandleFunc("/rest/config", configRESTHandler(cs))

	// create or present a download in the popup
	r.HandleFunc("/fetch", fetchHandler(cs, vm, dm))
	r.HandleFunc("/fetch/{id}", fetchHandler(cs, vm, dm))

	// get/update info on a download
	r.HandleFunc("/rest/fetch/{id}", fetchInfoOneRESTHandler(cs, dm))

	// version information
	r.HandleFunc("/rest/version", versionRESTHandler(vm))

	http.Handle("/", r)
	return r
}

// versionRESTHandler returns the version information, if we have up-to-date info from github
func versionRESTHandler(vm *version.Manager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if vm.GetInfo().GithubVersionFetched {
			b, _ := json.Marshal(vm.GetInfo())
			_, err := w.Write(b)
			if err != nil {
				log.Printf("could not write to client: %s", err)
			}
		} else {
			w.WriteHeader(400)
		}
	}
}

// homeHandler returns the main index page
func homeHandler(cs *config.ConfigService, vm *version.Manager, dm *download.Manager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		bookmarkletURL := fmt.Sprintf("javascript:(function(f,s,n,o){window.open(f+encodeURIComponent(s),n,o)}('%s/fetch?url=',window.location,'yourform','width=%d,height=%d'));", cs.Config.Server.Address, cs.Config.UI.PopupWidth, cs.Config.UI.PopupHeight)

		t, err := template.ParseFS(webFS, "data/templates/layout.tmpl", "data/templates/menu.tmpl", "data/templates/index.tmpl")
		if err != nil {
			panic(err)
		}

		type Info struct {
			Manager        *download.Manager
			BookmarkletURL template.URL
			Config         *config.Config
			Version        version.Info
		}

		info := Info{
			Manager:        dm,
			BookmarkletURL: template.URL(bookmarkletURL),
			Config:         cs.Config,
			Version:        vm.GetInfo(),
		}

		dm.Lock.Lock()
		defer dm.Lock.Unlock()
		err = t.ExecuteTemplate(w, "layout", info)
		if err != nil {
			panic(err)
		}
	}
}

// staticHandler handles requests for static files
func staticHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		filename := vars["filename"]
		if strings.Index(filename, ".js") == len(filename)-3 {
			f, err := webFS.Open("data/js/" + filename)
			if err != nil {
				log.Printf("error accessing %s - %v", filename, err)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, err = io.Copy(w, f)
			if err != nil {
				log.Printf("could not write to client: %s", err)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

// configHandler returns the configuration page
func configHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		t, err := template.ParseFS(webFS, "data/templates/layout.tmpl", "data/templates/menu.tmpl", "data/templates/config.tmpl")
		if err != nil {
			panic(err)
		}

		err = t.ExecuteTemplate(w, "layout", nil)
		if err != nil {
			panic(err)
		}
	}
}

// configRESTHandler handles both reading and writing of the configuration
func configRESTHandler(cs *config.ConfigService) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "POST" {
			log.Printf("Updating config")
			b, err := io.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			err = cs.Config.UpdateFromJSON(b)

			if err != nil {
				errorRes := errorResponse{Success: false, Error: err.Error()}
				errorResB, _ := json.Marshal(errorRes)
				w.WriteHeader(400)
				_, err = w.Write(errorResB)
				if err != nil {
					log.Printf("could not write to client: %s", err)
				}
				return
			}
			cs.WriteConfig()
		}
		b, _ := json.Marshal(cs.Config)
		_, err := w.Write(b)
		if err != nil {
			log.Printf("could not write config to client: %s", err)
		}
	}
}

func fetchInfoOneRESTHandler(cs *config.ConfigService, dm *download.Manager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idString := vars["id"]
		if idString != "" {
			id, err := strconv.Atoi(idString)
			if err != nil {
				http.NotFound(w, r)
				return
			}

			thisDownload, err := dm.GetDlById(id)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if thisDownload == nil {
				panic("should not happen")
			}

			if r.Method == "POST" {

				type updateRequest struct {
					Action string `json:"action"`
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
					_, err = w.Write(errorResB)
					if err != nil {
						log.Printf("could not write to client: %s", err)
					}
					return
				}

				if thisReq.Action == "stop" {

					thisDownload.Stop()
					succRes := successResponse{Success: true, Message: "download stopped"}
					succResB, _ := json.Marshal(succRes)
					_, err = w.Write(succResB)
					if err != nil {
						log.Printf("could not write to client: %s", err)
					}
					return
				}
			}

			// just a get, return the object
			thisDownload.Lock.Lock()
			defer thisDownload.Lock.Unlock()

			b, _ := json.Marshal(thisDownload)

			_, err = w.Write(b)
			if err != nil {
				log.Printf("could not write to client: %s", err)
			}
			return
		} else {
			http.NotFound(w, r)
		}
	}
}

func fetchInfoRESTHandler(dm *download.Manager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		b, err := dm.DownloadsAsJSON()
		if err != nil {
			panic(err)
		}
		_, err = w.Write(b)
		if err != nil {
			log.Printf("could not write to client: %s", err)
		}
	}
}

// fetchHandler shows the popup, either the initial form (for create) or the form when in
// progress (to be updated by REST) - this is determined by GET vs POST
func fetchHandler(cs *config.ConfigService, vm *version.Manager, dm *download.Manager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("fetchHandler ")

		method := r.Method

		// if they refreshed the popup, just load the existing object, don't
		// create a new one
		vars := mux.Vars(r)
		idString := vars["id"]

		idInt, idOK := strconv.ParseInt(idString, 10, 32)

		if method == "GET" && idOK == nil && idInt > 0 {
			// existing, load it up
			log.Printf("loading popup for id %d", idInt)
			dl, err := dm.GetDlById(int(idInt))
			if err != nil {
				log.Printf("not found")
				w.WriteHeader(404)
				return
			}

			t, err := template.ParseFS(webFS, "data/templates/layout.tmpl", "data/templates/popup.tmpl")
			if err != nil {
				panic(err)
			}

			templateData := map[string]interface{}{"dl": dl, "config": cs.Config, "canStop": download.CanStopDownload}

			err = t.ExecuteTemplate(w, "layout", templateData)
			if err != nil {
				panic(err)
			}
			return
		} else if method == "POST" {
			// creating a new one
			type reqType struct {
				URL               string `json:"url"`
				ProfileChosen     string `json:"profile"`
				DestinationChosen string `json:"destination"`
			}

			req := reqType{}
			json.NewDecoder(r.Body).Decode(&req)

			log.Printf("popup POST request: %#v", req)

			if req.URL == "" {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(errorResponse{
					Success: false,
					Error:   "No URL supplied",
				})
				return
			} else {

				if req.ProfileChosen == "" {

					w.WriteHeader(400)
					json.NewEncoder(w).Encode(errorResponse{
						Success: false,
						Error:   "you must choose a profile",
					})
					return
				}

				profile := cs.Config.ProfileCalled(req.ProfileChosen)
				if profile == nil {
					w.WriteHeader(400)
					json.NewEncoder(w).Encode(errorResponse{
						Success: false,
						Error:   fmt.Sprintf("no such profile: '%s'", req.ProfileChosen),
					})
					return
				}

				destination := cs.Config.DestinationCalled(req.DestinationChosen)
				if req.DestinationChosen != "" && destination == nil {
					w.WriteHeader(400)
					json.NewEncoder(w).Encode(errorResponse{
						Success: false,
						Error:   fmt.Sprintf("no such destination: '%s'", req.DestinationChosen),
					})
					return
				}

				// create the new download
				newDL := download.NewDownload(req.URL, cs.Config)
				id := newDL.Id
				newDL.Destination = destination
				newDL.DownloadProfile = *profile
				dm.AddDownload(newDL)
				dm.Queue(newDL)

				w.WriteHeader(200)
				json.NewEncoder(w).Encode(queuedResponse{
					Success:  true,
					Location: fmt.Sprintf("/fetch/%d", id),
				})
			}
		} else {
			// a GET, show the popup so they can start the download (or just close
			// the popup if they didn't mean it)
			log.Print("loading popup for a new download")
			query := r.URL.Query()
			url, present := query["url"]

			if !present {
				w.WriteHeader(400)
				fmt.Fprint(w, "No url supplied")
				return
			}

			t, err := template.ParseFS(webFS, "data/templates/layout.tmpl", "data/templates/popup_create.tmpl")
			if err != nil {
				panic(err)
			}
			templateData := map[string]interface{}{"config": cs.Config, "url": url[0]}

			err = t.ExecuteTemplate(w, "layout", templateData)
			if err != nil {
				panic(err)
			}

		}

	}
}
