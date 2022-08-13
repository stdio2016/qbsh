package qbsh

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func RandPitch(n int) []PitchType {
	out := make([]PitchType, n)
	for i := 0; i < n; i++ {
		out[i] = PitchType(rand.Intn(100))
	}
	return out
}

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

func TestSearch2(t *testing.T) {
	var d DTW_tmp
	for i := 1; i <= 10; i++ {
		for j := 1; j <= 10; j++ {
			rand.Seed(int64(i + j))
			query := RandPitch(i)
			song := MakeSong(RandPitch(j), "name")
			shift := PitchType(2)
			ans1 := DTW(song.Pitch, query, shift)
			ans2 := d.DTW_simd(song, query, 0, j, shift)
			fmt.Println(ans1, ans2)
			if ans1 != ans2 {
				t.Errorf("query %d song %d two method differ", i, j)
			}
		}
	}
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

func BenchmarkDTW(b *testing.B) {
	bytes, err := os.ReadFile("testdata/littlebee.txt")
	if err != nil {
		b.Error("test data not found!")
	}
	dat := string(bytes)
	pitch := ParsePitch(dat[:len(dat)-1])
	query := pitch[:128]
	for i := 0; i < b.N; i++ {
		DTW(pitch, query, 0)
	}
}

func BenchmarkDTWSimd(b *testing.B) {
	bytes, err := os.ReadFile("testdata/littlebee.txt")
	if err != nil {
		b.Error("test data not found!")
	}
	dat := string(bytes)
	pitch := ParsePitch(dat[:len(dat)-1])
	song := MakeSong(pitch, "")
	query := pitch[:128]
	var d DTW_tmp
	for i := 0; i < b.N; i++ {
		d.DTW_simd(song, query, 0, len(pitch), 0)
	}
}
