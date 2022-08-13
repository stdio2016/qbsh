#include "textflag.h"
DATA inf<>+0x00(SB)/4, $0x497423f0 // 999999.0
GLOBL inf<>(SB), (RODATA+NOPTR), $4

TEXT ·DTW_simd_impl(SB),$0-136
	// R0 = song
	MOVD song+0(FP), R0
	// R1 = query
	MOVD query+24(FP), R1
	// R2 = slen
	MOVD slen+48(FP), R2
	// R3 = qlen
	MOVD qlen+56(FP), R3
	// R4 = dp1
	MOVD dp1+64(FP), R4
	// R5 = dp2
	MOVD dp2+88(FP), R5
	// R6 = dp3
	MOVD dp3+112(FP), R6

	// both slen and qlen must > 0
	CMP ZR, R2
	BLE bad
	SUB ZR, R3
	BLE bad

	// initialize dp table
	// R7 = i-1 = 0
	// NOTE: arm64 has no address mode imm(Rn)(Rn<<imm)
	MOVD $1, R7
	// R8 = inf
	MOVW inf<>(SB), R8
	// F0 = best = inf
	FMOVS inf<>(SB), F0

init_dp:
	// dp1[i] = inf
	MOVW R8, (R4)(R7<<2)
	// dp2[i] = inf
	MOVW R8, (R5)(R7<<2)
	// dp3[i] = inf
	MOVW R8, (R6)(R7<<2)

	// i-1 < qlen
	ADD $1, R7, R7
	CMP R3, R7
	BLE init_dp

	// dp1[0] = 0
	MOVW ZR, (R4)
	
	// R7 = i
	// i := 1
	MOVD $1, R7
	// precompute &song[off] where off = slen
	ADD R2<<2, R0, R0
loop:
	// R8 = a
	// a := i - slen
	SUB R2, R7, R8
	// if a < 0 { a = 0 }
	CMP ZR, R8
	CSEL LT, ZR, R8, R8

	// R9 = b*4
	// b := i
	MOVD R7, R9
	// if b > qlen { b = qlen }
	CMP R3, R9
	CSEL GT, R3, R9, R9
	LSL $2, R9, R9

	// offset of song
	SUB $4, R0, R0

	// R8 = j*4, j = a already
	// R11 = R5 + 4
	// R12 = R6 + 4
	LSL $2, R8, R8
	ADD $4, R5, R11
	ADD $4, R6, R12
innerloop:
	// where R0 is &song[off]
	// F1 = song[off+j]
	// VLD1R (R0)(R8), [V1.S4]  or  ldr q1, [x0, x8]
	// but Golang doesnt allow VLD1R with offset and without post increment
	WORD $0x3CE86801
	// F2 = query[j]
	// VLD1R (R1)(R8), [V2.S4]  or  ldr q2, [x1, x8]
	WORD $0x3CE86822
	// F3 = v := dp2[j+1]
	// VLD1R (R11)(R8), [V3.S4]  or  ldr q3, [x11, x8]
	WORD $0x3CE86963
	// F4 = v2 := dp2[j]
	// VLD1R (R5)(R8), [V4.S4]  or  ldr q4, [x5, x8]
	WORD $0x3CE868A4
	// F5 = v3 := dp1[j]
	// VLD1R (R4)(R8), [V5.S4]  or  ldr q5, [x4, x8]
	WORD $0x3CE86885

	// F1 = diff := song[off+j] - query[j]
	// VFSUBS V2.S4, V1.S4, V1.S4 or fsub.4s v1, v1, v2
	WORD $0x4EA2D421

	// if v2 < v { v = v2 }
	// VFMINS V3.S4, V4.S4, V3.S4 or fmin.4s v3, v4, v3
	WORD $0x4EA3F483

	// if diff < 0 { diff = -diff }
	// VFABSS V1.S4, V1.S4 or fabs.4s v1, v1
	WORD $0x4EA0F821

	// if v3 < v { v = v3 }
	// VFMINS V3.S4, V5.S4, V3.S4 or fmin.4s v3, v5, v3
	WORD $0x4EA3F4A3

	// dp3[j+1] = v + diff
	// VFADDS V1.S4, V3.S4, V1.S4 or fadd.4s v1, v3, v1
	WORD $0x4E21D461
	
	// VST1R [V1.S4], (R12)(R8) or str q1, [x12, x8]
	WORD $0x3CA86981

	// j++
	ADD $16, R8, R8
	// j < b
	CMP R9, R8
	BLT innerloop

	// if dp3[qlen] < ans { ans = dp3[qlen] }
	FMOVS (R6)(R3<<2), F1
	FMINS F0, F1, F0

	// dp1, dp2, dp3 = dp2, dp3, dp1
	MOVD R4, R8
	MOVD R5, R4
	MOVD R6, R5
	MOVD R8, R6

	// i++
	ADD $1, R7, R7
	// i < slen+qlen
	ADD R2, R3, R8
	CMP R8, R7
	BLT loop

	FMOVS F0, r1+136(FP)
	RET

bad:
	MOVW inf<>(SB), R0
	MOVW R0, r1+136(FP)
	RET
