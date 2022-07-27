package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/stdio2016/qbsh"
)

func main() {
	flag.Parse()

	db := qbsh.InitDatabase()
	if len(flag.Args()) >= 1 {
		err := db.AddFromFile(flag.Arg(0))
		if err != nil {
			log.Default().Println("error while loading qbsh database")
			log.Default().Println(err)
		} else {
			log.Default().Printf("added %d songs\n", len(db.Songs))
		}
	}

	handleAdd := func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		songId := r.Form.Get("songId")
		name := r.Form.Get("name")
		s_pitch := r.Form.Get("pitch")
		if songId == "" {
			w.WriteHeader(400)
			fmt.Fprintf(w, "{\"error\":\"songId must not be empty\"}")
			return
		}
		pitch := qbsh.ParsePitch(s_pitch)
		song := qbsh.MakeSong(pitch, name)
		db.AddSong(song, songId)
		fmt.Fprintf(w, "{\"message\":\"Added song\"}")
		log.Default().Printf("Added song %s name %s\n", songId, name)
	}

	handleSearch := func(w http.ResponseWriter, r *http.Request) {
		s_pitch := r.URL.Query().Get("pitch")
		pitch := qbsh.ParsePitch(s_pitch)
		if len(pitch) == 0 {
			w.WriteHeader(400)
			fmt.Fprintf(w, "{\"error\":\"pitch must not be empty\"}")
			return
		}
		result := db.Search(pitch)
		b, _ := json.Marshal(result)
		w.Write(b)
		log.Default().Printf("Search song with pitch %v\n", pitch)
	}

	handleSearchLocalWav := func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("file")
		if filename == "" {
			result := qbsh.Result{
				Progress: "error",
				Reason:   "file must not be empty",
			}
			b, _ := json.Marshal(result)
			w.Write(b)
			return
		}
		time_1 := time.Now()
		pitch, err := qbsh.GetWavPitch(filename)
		if err != nil {
			result := qbsh.Result{
				Progress: "error",
				Reason:   err.Error(),
			}
			b, _ := json.Marshal(result)
			w.Write(b)
			return
		}
		pitch = qbsh.FixPitch(pitch)
		if len(pitch) == 0 {
			result := qbsh.Result{
				Progress: "error",
				Reason:   "Cannot analyze pitch. Maybe it is silent or full of noise.",
			}
			b, _ := json.Marshal(result)
			w.Write(b)
			return
		}
		time_2 := time.Now()
		result := db.Search(pitch)
		time_3 := time.Now()
		result.Reason = fmt.Sprintf("pitch %dms search %dms",
			time_2.Sub(time_1).Milliseconds(),
			time_3.Sub(time_2).Milliseconds())
		b, _ := json.Marshal(result)
		w.Write(b)
		log.Default().Printf("search local file %s\n", filename)
	}
	handlePing := func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "{\"status\":\"ok\"}")
	}

	http.HandleFunc("/add", handleAdd)
	http.HandleFunc("/search", handleSearch)
	http.HandleFunc("/searchLocalWav", handleSearchLocalWav)
	http.HandleFunc("/ping", handlePing)

	log.Default().Printf("Started server\n")
	http.ListenAndServe(":1606", nil)
}
