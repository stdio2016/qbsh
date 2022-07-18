package qbsh

import (
	"sort"
	"strconv"
	"strings"
)

func Median(arr []PitchType) PitchType {
	tmp := make([]PitchType, len(arr))
	copy(tmp, arr)
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i] < tmp[j]
	})
	mid := len(arr) / 2
	if len(arr)%2 == 0 {
		return (tmp[mid-1] + tmp[mid]) / 2
	}
	return tmp[mid]
}

func IntMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ParsePitch(line string) []PitchType {
	toks := strings.Split(line, " ")
	pitch := make([]PitchType, 0)
	for i := range toks {
		n, err := strconv.ParseFloat(toks[i], 64)
		if err != nil {
			continue
		}
		pitch = append(pitch, PitchType(n))
	}
	return pitch
}
