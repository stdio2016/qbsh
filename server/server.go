package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/stdio2016/qbsh"
)

func main() {
	db := qbsh.InitDatabase()

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
		pitch := []qbsh.PitchType{1, 2, 3}
		if len(pitch) == 0 {
			w.WriteHeader(400)
			fmt.Fprintf(w, "{\"error\":\"pitch must not be empty\"}")
			return
		}
		result := db.Search(pitch)
		b, _ := json.Marshal(result)
		w.Write(b)
		log.Default().Printf("TODO %s\n", filename)
	}

	http.HandleFunc("/add", handleAdd)
	http.HandleFunc("/search", handleSearch)
	http.HandleFunc("/searchLocalWav", handleSearchLocalWav)

	log.Default().Printf("Started server\n")
	http.ListenAndServe(":1606", nil)
}
