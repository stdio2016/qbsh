//copy from https://github.com/joaocarvalhoopen/Pitch_Detection_Algorithm
// Pitch Detection Algorithm
//
// YIN_algorithm.go - Reads a mono 16 bits wav  file and detects the main pitch of the
//                    audio, and the probability of it being correct.
//                    With this information you can detect the note that is played.
// The frequency detection algorithm is based on YIN algorithm more specifically
// a port to the Go ( GoLang ) programming language of the of the C implementation
// on https://github.com/ashokfernandez/Yin-Pitch-Tracking/blob/master/Yin.c
//
// Author:  Joao Nuno Carvalho
// Email:   joaonunocarv@gmail.com
// Date:    2017.12.9
// License: MIT OpenSource License
//
//package main
package qbsh

func findMainFrequency(buff []float32) (frequency float64, probability float64) {
	//arr :=  [YIN_SAMPLING_RATE / 2]float64{}
	//yin := Yin{0,0, arr, 0.0, 0.0}
	yin := Yin{}
	//bufferSize := 44100
	bufferSize := len(buff) // 11025
	threashold := 0.05
	yin.YinInit(bufferSize, threashold)
	frequency = yin.YinGetPitch(buff)
	probability = yin.YinGetProbability()
	return frequency, probability
}

//################################################################

const YIN_SAMPLING_RATE int = 44100
const YIN_DEFAULT_THRESHOLD float64 = 0.15
const BUFF_SIZE int = 11025 // 22050

type Yin struct {
	bufferSize     int       // Size of the buffer to process.
	halfBufferSize int       // Half of buffer size.
	yinBuffer      []float64 // Buffer that stores the results of the intermediate processing steps of the algorithm
	probability    float64   // Probability that the pitch found is correct as a decimal (i.e 0.85 is 85%)
	threshold      float64   // Allowed uncertainty in the result as a decimal (i.e 0.15 is 15%)
}

// threshold  Allowed uncertainty (e.g 0.05 will return a pitch with ~95% probability)
func (Y *Yin) YinInit(bufferSize int, threshold float64) {
	// Initialise the fields of the Yin structure passed in.
	Y.bufferSize = bufferSize
	Y.halfBufferSize = bufferSize / 2
	Y.probability = 0.0
	Y.threshold = threshold

	// Allocate the autocorellation buffer and initialise it to zero.
	Y.yinBuffer = make([]float64, BUFF_SIZE/2)
}

// Runs the Yin pitch detection algortihm
//        buffer       - Buffer of samples to analyse
// return pitchInHertz - Fundamental frequency of the signal in Hz. Returns -1 if pitch can't be found
func (Y *Yin) YinGetPitch(buffer []float32) (pitchInHertz float64) {
	//tauEstimate int      := -1
	pitchInHertz = -1

	// Step 1: CalcuYinGetPitchlates the squared difference of the signal with a shifted version of itself.
	Y.yinDifference(buffer)

	// Step 2: Calculate the cumulative mean on the normalised difference calculated in step 1.
	Y.yinCumulativeMeanNormalizedDifference()

	// Step 3: Search through the normalised cumulative mean array and find values that are over the threshold.
	tauEstimate := Y.yinAbsoluteThreshold()

	// Step 5: Interpolate the shift value (tau) to improve the pitch estimate.
	if tauEstimate != -1 {
		pitchInHertz = float64(YIN_SAMPLING_RATE) / Y.yinParabolicInterpolation(tauEstimate)
	}

	return pitchInHertz
}

// Certainty of the pitch found
// return ptobability - Returns the certainty of the note found as a decimal (i.e 0.3 is 30%)
func (Y *Yin) YinGetProbability() (probability float64) {
	return Y.probability
}

// Step 1: Calculates the squared difference of the signal with a shifted version of itself.
// @param buffer Buffer of samples to process.
//
// This is the Yin algorithms tweak on autocorellation. Read http://audition.ens.fr/adc/pdf/2002_JASA_YIN.pdf
// for more details on what is in here and why it's done this way.
func (Y *Yin) yinDifference(buffer []float32) {
	// Calculate the difference for difference shift values (tau) for the half of the samples.
	for tau := 0; tau < Y.halfBufferSize; tau++ {

		// Take the difference of the signal with a shifted version of itself, then square it.
		// (This is the Yin algorithm's tweak on autocorellation)
		for i := 0; i < Y.halfBufferSize; i++ {
			delta := float64(buffer[i]) - float64(buffer[i+tau])
			Y.yinBuffer[tau] += delta * delta
		}
	}
}

// Step 2: Calculate the cumulative mean on the normalised difference calculated in step 1
//
// This goes through the Yin autocorellation values and finds out roughly where shift is which
// produced the smallest difference
func (Y *Yin) yinCumulativeMeanNormalizedDifference() {
	runningSum := 0.0
	Y.yinBuffer[0] = 1

	// Sum all the values in the autocorellation buffer and nomalise the result, replacing
	// the value in the autocorellation buffer with a cumulative mean of the normalised difference.
	for tau := 1; tau < Y.halfBufferSize; tau++ {
		runningSum += Y.yinBuffer[tau]
		Y.yinBuffer[tau] *= float64(tau) / runningSum
	}
}

// Step 3: Search through the normalised cumulative mean array and find values that are over the threshold
// return Shift (tau) which caused the best approximate autocorellation. -1 if no suitable value is found
// over the threshold.
func (Y *Yin) yinAbsoluteThreshold() int {

	var tau int

	// Search through the array of cumulative mean values, and look for ones that are over the threshold
	// The first two positions in yinBuffer are always so start at the third (index 2)
	for tau = 2; tau < Y.halfBufferSize; tau++ {
		if Y.yinBuffer[tau] < Y.threshold {
			for (tau+1 < Y.halfBufferSize) && (Y.yinBuffer[tau+1] < Y.yinBuffer[tau]) {
				tau++
			}

			/* found tau, exit loop and return
			 * store the probability
			 * From the YIN paper: The yin->threshold determines the list of
			 * candidates admitted to the set, and can be interpreted as the
			 * proportion of aperiodic power tolerated
			 * within a periodic signal.
			 *
			 * Since we want the periodicity and and not aperiodicity:
			 * periodicity = 1 - aperiodicity */
			Y.probability = 1 - Y.yinBuffer[tau]
			break
		}
	}

	// if no pitch found, tau => -1
	if tau == Y.halfBufferSize || Y.yinBuffer[tau] >= Y.threshold {
		tau = -1
		Y.probability = 0
	}

	return tau
}

// Step 5: Interpolate the shift value (tau) to improve the pitch estimate.
// tauEstimate [description]
// Return
// The 'best' shift value for autocorellation is most likely not an interger shift of the signal.
// As we only autocorellated using integer shifts we should check that there isn't a better fractional
// shift value.
func (Y *Yin) yinParabolicInterpolation(tauEstimate int) float64 {

	var betterTau float64
	var x0 int
	var x2 int

	// Calculate the first polynomial coeffcient based on the current estimate of tau.
	if tauEstimate < 1 {
		x0 = tauEstimate
	} else {
		x0 = tauEstimate - 1
	}

	// Calculate the second polynomial coeffcient based on the current estimate of tau.
	if tauEstimate+1 < Y.halfBufferSize {
		x2 = tauEstimate + 1
	} else {
		x2 = tauEstimate
	}

	// Algorithm to parabolically interpolate the shift value tau to find a better estimate.
	if x0 == tauEstimate {
		if Y.yinBuffer[tauEstimate] <= Y.yinBuffer[x2] {
			betterTau = float64(tauEstimate)
		} else {
			betterTau = float64(x2)
		}
	} else if x2 == tauEstimate {
		if Y.yinBuffer[tauEstimate] <= Y.yinBuffer[x0] {
			betterTau = float64(tauEstimate)
		} else {
			betterTau = float64(x0)
		}
	} else {
		var s0, s1, s2 float64
		s0 = Y.yinBuffer[x0]
		s1 = Y.yinBuffer[tauEstimate]
		s2 = Y.yinBuffer[x2]
		// fixed AUBIO implementation, thanks to Karl Helgason:
		// (2.0f * s1 - s2 - s0) was incorrectly multiplied with -1
		betterTau = float64(tauEstimate) + (s2-s0)/(2*(2*s1-s2-s0))
	}

	return betterTau
}
