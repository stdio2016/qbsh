//go:build arm64

package qbsh

const DTW_simd_has_impl = true
const DTW_simd_width = 4

func DTW_simd_impl(song, query []PitchType, slen, qlen int, dp1, dp2, dp3 []PitchType) PitchType
