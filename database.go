package qbsh

import (
	"math"
	"os"
	"sort"
	"strings"
	"sync"
)

type PitchType float32

type Song struct {
	Name         string
	Pitch        []PitchType
	Artist       string
	Median       PitchType
	Low          PitchType
	High         PitchType
	Ranges       []SongPitchRange
	PitchForSimd []PitchType
}

type SongPitchRange struct {
	From   int
	To     int
	Median PitchType
}

type Database struct {
	Songs map[string]*Song
	Lock  sync.RWMutex
}

type SongScore struct {
	SongId string    `json:"file"`
	Name   string    `json:"name"`
	Score  PitchType `json:"score"`
	Artist string    `json:"singer"`
}

type Result struct {
	// progress must be "100" to indicate success
	// or "error" to indicate error
	Progress string      `json:"progress"`
	Pitch    []PitchType `json:"pitch"`
	Songs    []SongScore `json:"songs"`
	Reason   string      `json:"reason"`
}

func InitDatabase() *Database {
	return &Database{
		Songs: make(map[string]*Song),
	}
}

func (db *Database) AddFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	str := string(data)
	lines := strings.Split(str, "\n")
	for i := 0; i < len(lines)/4; i++ {
		songId := lines[i*4]
		name := lines[i*4+1]
		artist := lines[i*4+2]
		pitch := ParsePitch(lines[i*4+3])
		song := MakeSong(pitch, name)
		song.Artist = artist
		db.AddSong(song, songId)
	}
	return nil
}

func (db *Database) AddSong(song *Song, id string) {
	db.Lock.Lock()
	// I do not allow empty song in database
	if len(song.Pitch) == 0 {
		delete(db.Songs, id)
	} else {
		db.Songs[id] = song
	}
	db.Lock.Unlock()
}

func (db *Database) Search(query []PitchType) Result {
	q_mi := Median(query)

	var d DTW_tmp
	db.Lock.RLock()
	result := make([]SongScore, len(db.Songs))
	i := 0
	for songId, song := range db.Songs {
		best := PitchType(99999.0)
		songName := song.Name
		for _, ran := range song.Ranges {
			sco := d.DTW_simd(song, query, ran.From, ran.To, q_mi-ran.Median)
			if sco < best {
				best = sco
			}
		}
		result[i] = SongScore{songId, songName, best, song.Artist}
		i++
	}
	db.Lock.RUnlock()

	sort.Slice(result, func(i, j int) bool {
		return result[i].Score < result[j].Score
	})

	return Result{
		Progress: "100",
		Pitch:    query,
		Songs:    result[:IntMin(10, len(result))],
	}
	/*for rank, sco := range result[:IntMin(10, len(result))] {
		fmt.Printf("%d. %s %f\n", rank+1, sco.Name, sco.Score)
	}*/
}

func MakeSong(pitch []PitchType, name string) *Song {
	var med PitchType
	if len(pitch) > 0 {
		med = Median(pitch)
	}
	lo := med - 2
	hi := med + 2
	for i := 0; i < len(pitch)-128; i += 16 {
		med := Median(pitch[i : i+127])
		if med < lo {
			lo = med
		}
		if med > hi {
			hi = med
		}
	}

	var cum float64
	cumsums := []float64{0}
	for i := 0; i < len(pitch); i++ {
		cum += float64(pitch[i])
		cumsums = append(cumsums, cum)
	}
	upper := make([]int, len(pitch))
	lower := make([]int, len(pitch))
	isInit := make([]bool, len(pitch))
	for i := 0; i <= len(pitch)-80; i++ {
		mymax := 0
		mymin := 0
		first := true
		for j := IntMin(len(pitch)-i, 240); j > 0; j-- {
			if j >= 80 {
				subavg := int(math.Round((cumsums[i+j] - cumsums[i]) / float64(j)))
				if first || subavg > mymax {
					mymax = subavg
				}
				if first || subavg < mymin {
					mymin = subavg
				}
			}
			first = false
			if !isInit[i+j-1] || mymax > upper[i+j-1] {
				upper[i+j-1] = mymax
			}
			if !isInit[i+j-1] || mymin < lower[i+j-1] {
				isInit[i+j-1] = true
				lower[i+j-1] = mymin
			}
		}
	}
	var ranges []SongPitchRange
	if len(pitch) >= 80 {
		for trans := int(lo); trans <= int(hi); trans++ {
			flag := false
			from := 0
			for i := range upper {
				if lower[i]-1 <= trans && trans <= upper[i]+1 {
					if !flag {
						from = i
					}
					flag = true
				} else {
					if flag {
						r := SongPitchRange{from, i, PitchType(trans)}
						ranges = append(ranges, r)
					}
					flag = false
				}
			}
			if flag {
				r := SongPitchRange{from, len(upper), PitchType(trans)}
				ranges = append(ranges, r)
			}
		}
	}
	process_pitch := ProcessSongForSimd(pitch)
	return &Song{
		Name:         name,
		Pitch:        pitch,
		Median:       med,
		Low:          lo,
		High:         hi,
		Ranges:       ranges,
		PitchForSimd: process_pitch,
	}
}

func ProcessSongForSimd(pitch []PitchType) []PitchType {
	// reverse song pitch, then zero pad by 8
	out := make([]PitchType, len(pitch)+8)
	for i := range pitch {
		out[len(pitch)-1-i] = pitch[i]
	}
	return out
}

func DTW(song []PitchType, query []PitchType, shift PitchType) PitchType {
	n1 := len(song)
	n2 := len(query)
	dp1 := make([]PitchType, n2+1)
	for i := 1; i <= n2; i++ {
		dp1[i] = PitchType(999999.0)
	}
	dp2 := make([]PitchType, n2+1)
	ans := PitchType(999999.0)
	for i := 0; i < n1; i++ {
		for j := 0; j < n2; j++ {
			diff := song[i] + shift - query[j]
			if diff < 0 {
				diff = -diff
			}
			v := dp1[j+1]
			v2 := dp1[j]
			v3 := dp2[j]
			if v2 < v {
				v = v2
			}
			if v3 < v {
				v = v3
			}
			dp2[j+1] = v + diff
		}
		dp1, dp2 = dp2, dp1
		if dp1[n2] < ans {
			ans = dp1[n2]
		}
	}
	return ans
}
