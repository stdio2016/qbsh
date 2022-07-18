package qbsh

import (
	"sort"
	"sync"
)

type PitchType float32

type Song struct {
	Name   string
	Pitch  []PitchType
	Median PitchType
	Low    PitchType
	High   PitchType
}

type Database struct {
	Songs map[string]*Song
	Lock  sync.RWMutex
}

type SongScore struct {
	SongId string
	Name   string
	Score  PitchType
}

func InitDatabase() *Database {
	return &Database{
		Songs: make(map[string]*Song),
	}
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

func (db *Database) Search(query []PitchType) {
	result := make([]SongScore, 0)
	q_mi := Median(query)

	db.Lock.RLock()
	for songId, song := range db.Songs {
		best := PitchType(99999.0)
		songName := song.Name
		for t := song.Low; t <= song.High; t++ {
			sco := DTW(song.Pitch, query, q_mi-PitchType(t))
			if sco < best {
				best = sco
			}
		}
		result = append(result, SongScore{songId, songName, best})
	}
	db.Lock.RUnlock()

	sort.Slice(result, func(i, j int) bool {
		return result[i].Score < result[j].Score
	})

	/*for rank, sco := range result[:IntMin(10, len(result))] {
		fmt.Printf("%d. %s %f\n", rank+1, sco.Name, sco.Score)
	}*/
}

func MakeSong(pitch []PitchType, name string) *Song {
	med := Median(pitch)
	lo := med - 2
	hi := med + 2
	for i := 0; i < len(pitch)-128; i += 16 {
		med := Median(pitch[i : i+128])
		if med < lo {
			lo = med
		}
		if med > hi {
			hi = med
		}
	}
	return &Song{
		Name:   name,
		Pitch:  pitch,
		Median: med,
		Low:    lo,
		High:   hi,
	}
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
