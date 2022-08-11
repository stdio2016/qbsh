package qbsh

const DTW_simd_has_impl = true
const DTW_simd_width = 1

// a dummy implementation but it is required
func (d *DTW_tmp) DTW_simd_impl(song []PitchType, slen int) PitchType {
	const inf PitchType = 999999
	dp1 := d.Dp1
	dp2 := d.Dp2
	dp3 := d.Dp3
	qlen := d.Qlen
	query := d.Query

	ans := inf

	// initialize dp table
	for i := 1; i <= qlen; i++ {
		dp1[i] = inf
		dp2[i] = inf
		dp3[i] = inf
	}

	// search from any starting point
	dp1[0] = 0

	// diagonal
	for i := 1; i < slen+qlen; i++ {
		a := i - slen
		if a < 0 {
			a = 0
		}
		b := i
		if b > qlen {
			b = qlen
		}
		off := slen - i
		for j := a; j < b; j++ {
			diff := song[off+j] - query[j]
			if diff < 0 {
				diff = -diff
			}
			v := dp2[j+1]
			v2 := dp2[j]
			v3 := dp1[j]
			if v2 < v {
				v = v2
			}
			if v3 < v {
				v = v3
			}
			dp3[j+1] = v + diff
		}
		if dp3[qlen] < ans {
			ans = dp3[qlen]
		}
		dp1, dp2, dp3 = dp2, dp3, dp1
	}
	return ans
}
