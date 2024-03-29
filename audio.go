package qbsh

import (
	"math"

	"github.com/unixpickle/wav"
)

func ConvertHzToPitch(frequency float64) PitchType {
	if frequency <= 0 {
		return -1
	}
	midiPitch := math.Log2(frequency/440.0)*12.0 + 69.0
	out := math.Round(midiPitch*10) / 10.0
	if out < 0 {
		out = 0
	}
	return PitchType(out)
}

func GetWavPitch(path string) ([]PitchType, error) {
	s, err := wav.ReadSoundFile(path)
	if err != nil {
		return nil, err
	}

	bufSize := 2048
	stepSize := s.SampleRate() / 16
	buf := make([]float32, bufSize)
	out := make([]PitchType, 0)
	j := 0
	for _, sample := range s.Samples() {
		if j >= 0 {
			buf[j] = float32(sample)
		}
		j++
		if j == bufSize {
			frequency, probability := findMainFrequency(buf)
			frequency *= float64(s.SampleRate()) / 44100
			pitch := PitchType(-1)
			if probability > 0.5 {
				pitch = ConvertHzToPitch(frequency)
			}
			out = append(out, pitch)
			// move buffer
			for i := 0; i < bufSize-stepSize; i++ {
				buf[i] = buf[i+stepSize]
			}
			j -= stepSize
		}
	}
	return out, nil
}

func GetWavPitch2(path string) ([]PitchType, error) {
	s, err := wav.ReadSoundFile(path)
	if err != nil {
		return nil, err
	}

	bufSize := 512
	for bufSize < s.SampleRate()/30 {
		bufSize *= 2
	}
	stepSize := s.SampleRate() / 100
	buf := make([]float64, bufSize)
	pyin := PyinCreate(bufSize, s.SampleRate())
	pyin.HopLength = stepSize
	pyin.PyinInit()
	out := make([]PitchType, 0)
	j := 0
	prob := pyin.PyinHMMInit()
	backpath := make([][]int, 0)
	frames := make([][]PyinCandidate, 0)
	for _, sample := range s.Samples() {
		if j >= 0 {
			buf[j] = float64(sample)
		}
		j++
		if j == bufSize {
			cand := pyin.PyinFindFrequency(buf)
			var back []int
			prob, back = pyin.PyinHMMForward(cand, prob)
			backpath = append(backpath, back)
			frames = append(frames, cand)
			frequency := -1.0
			if len(cand) > 0 {
				frequency = cand[0].Frequency
			}
			probability := 0.0
			for i := range cand {
				probability += cand[i].Probability
			}
			pitch := PitchType(-1)
			if probability > 0.5 {
				pitch = ConvertHzToPitch(frequency)
			}
			out = append(out, pitch)
			// move buffer
			for i := 0; i < bufSize-stepSize; i++ {
				buf[i] = buf[i+stepSize]
			}
			j -= stepSize
		}
	}
	better := pyin.PyinHMMViterbi(frames, backpath, prob)
	for i := range better {
		out[i] = ConvertHzToPitch(better[i].Frequency)
		if better[i].Probability < 0.3 {
			out[i] = -1
		}
	}
	resample := make([]PitchType, len(out)/5)
	for i := 0; (i+1)*5 <= len(out); i++ {
		resample[i] = Median(out[i*5 : (i+1)*5])
	}
	return resample, nil
}

func FixPitch(pitchVec []PitchType) []PitchType {
	// crop trailing silence
	start := 0
	for ; start < len(pitchVec); start++ {
		if pitchVec[start] != -1 {
			break
		}
	}
	end := len(pitchVec) - 1
	for ; end >= 0; end-- {
		if pitchVec[end] != -1 {
			break
		}
	}
	if start > end {
		return nil
	}
	pitchVec = pitchVec[start : end+1]
	ptc := make([]PitchType, len(pitchVec))
	copy(ptc, pitchVec)

	// fill missing pitch
	prevPitch := PitchType(0)
	for i := range ptc {
		if ptc[i] == -1 {
			ptc[i] = prevPitch
		} else {
			prevPitch = ptc[i]
		}
	}

	// remove extreme value
	/*mid := Median(ptc)
	for i := range ptc {
		if ptc[i]-mid >= 12 || mid-ptc[i] >= 12 {
			ptc[i] = mid + PitchType(math.Mod(float64(ptc[i]-mid), 12.0))
		}
	}*/

	// median filter
	ptc2 := make([]PitchType, len(ptc))
	copy(ptc2, ptc)
	for i := 1; i < len(ptc)-1; i++ {
		a := ptc[i-1]
		b := ptc[i]
		c := ptc[i+1]
		if a < b {
			a, b = b, a
		}
		if c > a {
			ptc2[i] = a
		} else if c > b {
			ptc2[i] = c
		} else {
			ptc2[i] = b
		}
	}
	return ptc2
}
