package qbsh

type DTW_tmp struct {
	Qlen  int
	Query []PitchType
	Dp1   []PitchType
	Dp2   []PitchType
	Dp3   []PitchType
}

func (d *DTW_tmp) DTW_simd(song *Song, query []PitchType, from, to int, shift PitchType) PitchType {
	if DTW_simd_has_impl {
		// fill the blank
		need_size := len(query) + DTW_simd_width
		if len(d.Query) < need_size {
			d.Query = make([]PitchType, need_size)
		}
		if len(d.Dp1) < need_size {
			d.Dp1 = make([]PitchType, need_size)
		}
		if len(d.Dp2) < need_size {
			d.Dp2 = make([]PitchType, need_size)
		}
		if len(d.Dp3) < need_size {
			d.Dp3 = make([]PitchType, need_size)
		}

		for i := range query {
			d.Query[i] = query[i] - shift
		}
		d.Qlen = len(query)
		nSong := len(song.Pitch)
		ans := DTW_simd_impl(song.PitchForSimd[nSong-to:nSong-from], d.Query, nSong, d.Qlen, d.Dp1, d.Dp2, d.Dp3)
		return ans
	}
	// return good old implementation
	return DTW(song.Pitch[from:to], query, shift)
}
