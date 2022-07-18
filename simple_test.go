package qbsh

import (
	"os"
	"testing"
)

func TestAdd(t *testing.T) {
	db := InitDatabase()
	song := MakeSong([]PitchType{1, 2, 3}, "1")
	db.AddSong(song, "1")
	if _, ok := db.Songs["1"]; !ok {
		t.Error("Song 1 not found!")
	}
}

func TestSearch1(t *testing.T) {
	db := InitDatabase()
	db.AddSong(MakeSong([]PitchType{1, 2, 3}, "SongA"), "1")
	db.AddSong(MakeSong([]PitchType{3, 2, 1}, "SongB"), "2")
	query := []PitchType{1, 2, 3}
	db.Search(query)
}

func BenchmarkSearch(b *testing.B) {
	bytes, err := os.ReadFile("testdata/littlebee.txt")
	if err != nil {
		b.Error("test data not found!")
	}
	dat := string(bytes)
	pitch := ParsePitch(dat[:len(dat)-1])
	db := InitDatabase()
	littlebee := MakeSong(pitch, "little bee")
	db.AddSong(littlebee, "1")
	littlebee2 := MakeSong(pitch, "little bee")
	db.AddSong(littlebee2, "2")
	query := pitch[:128]
	for i := 0; i < b.N; i++ {
		db.Search(query)
	}
}
