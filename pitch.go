package qbsh

import (
	"math"
	"sort"

	"github.com/mjibson/go-dsp/fft"
)

type BetaParameters struct {
	Alpha float64
	Beta  float64
}

type Pyin struct {
	// parameters
	Fmin              float64
	Fmax              float64
	Sr                int
	FrameLength       int
	HopLength         int
	NThresholds       int
	BetaParameters    BetaParameters
	Resolution        float64
	MaxTransitionRate float64
	SwitchProb        float64
	NoTroughProb      float64

	// buffers
	Buffer          []complex128
	D               []float64
	Prior           []float64
	Transition      [][]float64
	TransitionStart []int
	NPitchBins      int
}

type PyinCandidate struct {
	Frequency   float64
	Probability float64
}

func PyinCreate(framesize int, samplerate int) *Pyin {
	return &Pyin{
		Fmin:              55,
		Fmax:              1047,
		Sr:                samplerate,
		FrameLength:       framesize,
		HopLength:         framesize / 4,
		NThresholds:       100,
		BetaParameters:    BetaParameters{2, 11 + 1.0/3},
		Resolution:        0.1,
		MaxTransitionRate: 35.92,
		SwitchProb:        0.01,
		NoTroughProb:      0.01,
	}
}

func (pyin *Pyin) PyinInit() {
	pyin.NPitchBins = int(12*math.Log2(pyin.Fmax/pyin.Fmin)/pyin.Resolution) + 1

	// beta distribution
	pyin.Prior = make([]float64, pyin.NThresholds)
	alpha := pyin.BetaParameters.Alpha
	beta := pyin.BetaParameters.Beta
	psum := 0.0
	for i := range pyin.Prior {
		x := float64(i+1) / float64(pyin.NThresholds)
		pyin.Prior[i] = math.Pow(x, alpha-1) * math.Pow(1-x, beta-1)
		psum += pyin.Prior[i]
	}
	for i := range pyin.Prior {
		pyin.Prior[i] /= psum
	}

	// HMM transition table
	maxStep := pyin.MaxTransitionRate * 12 * float64(pyin.HopLength) / float64(pyin.Sr)
	pyin.Transition = make([][]float64, pyin.NPitchBins*2)
	pyin.TransitionStart = make([]int, pyin.NPitchBins*2)
	for i := range pyin.Transition {
		p := i / 2
		a := IntMax(p-int(maxStep), 0)
		b := IntMin(p+int(maxStep), pyin.NPitchBins-1)
		pyin.Transition[i] = make([]float64, (b-a+1)*2)
		pyin.TransitionStart[i] = a * 2

		voicedProb := 1 - pyin.SwitchProb
		unvoicedProb := pyin.SwitchProb
		if i&1 == 1 {
			voicedProb = pyin.SwitchProb
			unvoicedProb = 1 - pyin.SwitchProb
		}
		for j := a; j <= b; j++ {
			pitchProb := maxStep - math.Abs(float64(j-p))
			idx := j - a
			pyin.Transition[i][idx*2] = voicedProb * pitchProb
			pyin.Transition[i][idx*2+1] = unvoicedProb * pitchProb
		}
		// normalize probability
		sum := 0.0
		for _, x := range pyin.Transition[i] {
			sum += x
		}
		for j := range pyin.Transition[i] {
			pyin.Transition[i][j] /= sum
		}
	}

	// initialize buffer
	pyin.Buffer = make([]complex128, pyin.FrameLength*2)
	pyin.D = make([]float64, pyin.FrameLength)
}

func (pyin *Pyin) PyinFindFrequency(buffer []float64) []PyinCandidate {
	pyin.pyinDifference(buffer)
	pyin.pyinCumulativeMean()
	return pyin.pyinFindDip()
}

func (pyin *Pyin) pyinDifference(buffer []float64) {
	n := len(buffer)
	x := pyin.Buffer
	for i := range buffer {
		x[i] = complex(buffer[i], 0)
	}
	x = fft.FFT(x)
	// x = x * conj(x)
	for i := range x {
		re := real(x[i])
		im := imag(x[i])
		x[i] = complex(re*re+im*im, 0)
	}
	x = fft.IFFT(x)
	// now compute sum of x^2
	y := pyin.D
	sum := 0.0
	for i := range buffer {
		sum += buffer[i] * buffer[i]
		y[n-1-i] = sum
	}
	partsum := 0.0
	for i := range buffer {
		y[i] = y[i] + (sum - partsum) - 2*real(x[i])
		partsum += buffer[i] * buffer[i]
	}
}

// it modifies input in place
func (pyin *Pyin) pyinCumulativeMean() {
	partsum := 0.0
	pyin.D[0] = 1
	for i := 1; i < len(pyin.D); i++ {
		partsum += pyin.D[i]
		pyin.D[i] = pyin.D[i] * float64(i) / partsum
	}
}

func (pyin *Pyin) pyinFindDip() []PyinCandidate {
	dips := make([]PyinCandidate, 0)
	thres := pyin.NThresholds
	argminD := 2.0
	minD := 1.0
	minPeriod := int(float64(pyin.Sr) / pyin.Fmax)
	minPeriod = IntMax(minPeriod, 2)
	maxPeriod := int(float64(pyin.Sr)/pyin.Fmin) + 1
	maxPeriod = IntMin(maxPeriod, pyin.FrameLength-1)
	for tau := minPeriod; tau < maxPeriod; tau++ {
		d := pyin.D[tau]
		if d < pyin.D[tau-1] && d <= pyin.D[tau+1] {
			// found a dip
			better_tau := pyin.pyinParabolicInterpolation(tau)
			freq := float64(pyin.Sr) / better_tau
			candidate := PyinCandidate{
				Frequency: freq,
			}
			if d < minD {
				argminD = freq
				minD = d
			}
			for thres > 0 && d < float64(thres)/float64(pyin.NThresholds) {
				candidate.Probability += pyin.Prior[thres-1]
				thres--
			}
			if candidate.Probability > 0 {
				dips = append(dips, candidate)
			}
		}
	}
	lastProb := 0.0
	for i := 0; i < thres; i++ {
		lastProb += pyin.Prior[i]
	}
	lastProb *= pyin.NoTroughProb
	if len(dips) > 0 && dips[len(dips)-1].Frequency == argminD {
		dips[len(dips)-1].Probability += lastProb
	} else if lastProb > 0 {
		dips = append(dips, PyinCandidate{
			Frequency:   argminD,
			Probability: lastProb,
		})
	}
	sort.Slice(dips, func(i, j int) bool {
		return dips[i].Probability > dips[j].Probability
	})
	return dips
}

func (pyin *Pyin) pyinParabolicInterpolation(pos int) float64 {
	t0 := pos - 1
	t2 := pos + 1
	s0 := pyin.D[t0]
	s1 := pyin.D[pos]
	s2 := pyin.D[t2]
	return float64(pos) - (s2-s0)/((s2+s0-s1*2)*2)
}

func (pyin *Pyin) PyinHMMInit() []float64 {
	out := make([]float64, pyin.NPitchBins*2)
	for i := range out {
		out[i] = 1.0 / float64(len(out))
	}
	return out
}

func (pyin *Pyin) pyinNearestFrequencyBin(frame []PyinCandidate) []int {
	nearest := make([]int, pyin.NPitchBins)
	freqs := make([]float64, len(frame))
	for i := range frame {
		freq := math.Log2(frame[i].Frequency/pyin.Fmin) * 12 / pyin.Resolution
		freqs[i] = math.Abs(freq - math.Round(freq))
		freqBin := int(math.Round(freq))
		if freqBin < 0 {
			freqBin = 0
		} else if freqBin >= pyin.NPitchBins {
			freqBin = pyin.NPitchBins - 1
		}
		if nearest[freqBin] > 0 {
			if freqs[i] < freqs[nearest[freqBin]-1] {
				nearest[freqBin] = i + 1
			}
		} else {
			nearest[freqBin] = i + 1
		}
	}
	return nearest
}

func (pyin *Pyin) PyinHMMForward(frame []PyinCandidate, prob []float64) ([]float64, []int) {
	newprob := make([]float64, pyin.NPitchBins*2)
	path := make([]int, len(newprob))
	for i := range prob {
		jbase := pyin.TransitionStart[i]
		trans := pyin.Transition[i]
		for j := range trans {
			nxtProb := prob[i] * trans[j]
			if nxtProb > newprob[jbase+j] {
				newprob[jbase+j] = nxtProb
				path[jbase+j] = i
			}
		}
	}

	// voiced probability
	nearest := pyin.pyinNearestFrequencyBin(frame)
	voicedProb := 0.0
	for i := range nearest {
		if nearest[i] > 0 {
			voicedProb += frame[nearest[i]-1].Probability
		}
	}

	// calculate observe probability
	for i := range newprob {
		var p float64
		if i&1 == 0 {
			// voiced
			if nearest[i/2] > 0 {
				p = frame[nearest[i/2]-1].Probability
			}
		} else {
			// unvoiced
			p = (1 - voicedProb) / float64(pyin.NPitchBins)
		}
		newprob[i] *= p
	}

	// normalize probability
	sum := 0.0
	for i := range newprob {
		sum += newprob[i]
	}
	if sum == 0.0 {
		// probability sum to 0?
		for i := range newprob {
			newprob[i] = 1
			path[i] = i
		}
	} else {
		for i := range newprob {
			newprob[i] /= sum
		}
	}
	return newprob, path
}

func (pyin *Pyin) PyinHMMViterbi(frame [][]PyinCandidate, path [][]int, finalProb []float64) []PyinCandidate {
	out := make([]PyinCandidate, len(frame))
	state := 0
	for i := range finalProb {
		if finalProb[i] > finalProb[state] {
			state = i
		}
	}
	for i := len(frame) - 1; i >= 0; i-- {
		nearest := pyin.pyinNearestFrequencyBin(frame[i])
		if nearest[state/2] > 0 {
			out[i] = frame[i][nearest[state/2]-1]
		} else if len(frame[i]) > 0 {
			// rescue frequency
			var minDiff float64
			var which int
			first := true
			for j := range frame[i] {
				freq := math.Log2(frame[i][j].Frequency/pyin.Fmin) * 12 / pyin.Resolution
				freqDiff := math.Abs(freq - math.Round(freq))
				if freqDiff < minDiff || first {
					minDiff = freqDiff
					which = j
				}
				first = false
			}
			out[i] = frame[i][which]
		} else {
			// no pitch detected but in voice state?
			out[i].Frequency = -1
		}
		if state&1 == 1 { // unvoiced
			out[i].Frequency = -1
		}
		state = path[i][state]
	}
	return out
}
