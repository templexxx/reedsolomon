// Reference: www.ssrc.ucsc.edu/Papers/plank-fast13.pdf

#include "textflag.h"

#define low_tblx X0
#define high_tblx X1
#define low_tbly Y0
#define high_tbly Y1
#define low_tbl Z0
#define high_tbl Z1

#define maskx X2
#define masky Y2
#define mask Z2

#define in0x  X3
#define in0y  Y3
#define in0  Z3
#define in1  Z4
#define in2  Z5
#define in3  Z6
#define in4  Z7
#define in5  Z8

#define in0_hx  X10
#define in0_hy  Y10
#define in0_h  Z10
#define in1_h  Z11
#define in2_h  Z12
#define in3_h  Z13
#define in4_h  Z14
#define in5_h  Z15

#define in  BX
#define out DI
#define len R8
#define pos R9

#define tmp0 R10

// func coeffMulVectAVX512(tbl, d, p []byte)
TEXT ·coeffMulVectAVX512(SB), NOSPLIT, $0
	MOVQ         i+24(FP), in
	MOVQ         o+48(FP), out
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), low_tblx
	VMOVDQU      16(tmp0), high_tblx
	VINSERTI128  $1, low_tblx, low_tbly, low_tbly
	VINSERTI128  $1, high_tblx, high_tbly, high_tbly
	VINSERTI64X4 $1, low_tbly, low_tbl, low_tbl
	VINSERTI64X4 $1, high_tbly, high_tbl, high_tbl
	MOVB         $0x0f, DX
	VPBROADCASTB DX, mask
	MOVQ         in_len+32(FP), len
	TESTQ        $31, len
	JNZ          one16b

big:
	TESTQ $511, len
	JNZ   not_aligned

	// 256bytes/loop
aligned:
	MOVQ $0, pos

loop256b:
	// split low/high part(every byte get 4 low/high bit)
	VMOVDQU8 (in)(pos*1), in0
	VPSRLQ   $4, in0, in0_h
	VPANDQ   mask, in0_h, in0_h
	VPANDQ   mask, in0, in0

	// according low/high part shuffle table, store result in second dst register
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in0, low_tbl, in0
	VPXORQ  in0, in0_h, in0

	// store result in memory
	VMOVDQU8 in0, (out)(pos*1)

	VMOVDQU8 64(in)(pos*1), in1
	VPSRLQ   $4, in1, in1_h
	VPANDQ   mask, in1_h, in1_h
	VPANDQ   mask, in1, in1
	VPSHUFB  in1_h, high_tbl, in1_h
	VPSHUFB  in1, low_tbl, in1
	VPXORQ   in1, in1_h, in1
	VMOVDQU8 in1, 64(out)(pos*1)

	VMOVDQU8 128(in)(pos*1), in2
	VPSRLQ   $4, in2, in2_h
	VPANDQ   mask, in2_h, in2_h
	VPANDQ   mask, in2, in2
	VPSHUFB  in2_h, high_tbl, in2_h
	VPSHUFB  in2, low_tbl, in2
	VPXORQ   in2, in2_h, in2
	VMOVDQU8 in2, 128(out)(pos*1)

	VMOVDQU8 192(in)(pos*1), in3
	VPSRLQ   $4, in3, in3_h
	VPANDQ   mask, in3_h, in3_h
	VPANDQ   mask, in3, in3
	VPSHUFB  in3_h, high_tbl, in3_h
	VPSHUFB  in3, low_tbl, in3
	VPXORQ   in3, in3_h, in3
	VMOVDQU8 in3, 192(out)(pos*1)

	VMOVDQU8 256(in)(pos*1), in4
	VPSRLQ   $4, in4, in4_h
	VPANDQ   mask, in4_h, in4_h
	VPANDQ   mask, in4, in4
	VPSHUFB  in4_h, high_tbl, in4_h
	VPSHUFB  in4, low_tbl, in4
	VPXORQ   in4, in4_h, in4
	VMOVDQU8 in4, 256(out)(pos*1)

	VMOVDQU8 320(in)(pos*1), in5
	VPSRLQ   $4, in5, in5_h
	VPANDQ   mask, in5_h, in5_h
	VPANDQ   mask, in5, in5
	VPSHUFB  in5_h, high_tbl, in5_h
	VPSHUFB  in5, low_tbl, in5
	VPXORQ   in5, in5_h, in5
	VMOVDQU8 in5, 320(out)(pos*1)

	VMOVDQU8 384(in)(pos*1), in0
	VPSRLQ   $4, in0, in0_h
	VPANDQ   mask, in0_h, in0_h
	VPANDQ   mask, in0, in0
	VPSHUFB  in0_h, high_tbl, in0_h
	VPSHUFB  in0, low_tbl, in0
	VPXORQ   in0, in0_h, in0
	VMOVDQU8 in0, 384(out)(pos*1)

	VMOVDQU8 448(in)(pos*1), in1
	VPSRLQ   $4, in1, in1_h
	VPANDQ   mask, in1_h, in1_h
	VPANDQ   mask, in1, in1
	VPSHUFB  in1_h, high_tbl, in1_h
	VPSHUFB  in1, low_tbl, in1
	VPXORQ   in1, in1_h, in1
	VMOVDQU8 in1, 448(out)(pos*1)

	ADDQ $512, pos
	CMPQ len, pos
	JNE  loop256b
	VZEROUPPER
	RET

not_aligned:
	TESTQ $63, len
	JNZ   one32b
	MOVQ  len, tmp0
	ANDQ  $511, tmp0

loop64b:
	VMOVDQU8 -64(in)(len*1), in0
	VPSRLQ   $4, in0, in0_h
	VPANDQ   mask, in0_h, in0_h
	VPANDQ   mask, in0, in0
	VPSHUFB  in0_h, high_tbl, in0_h
	VPSHUFB  in0, low_tbl, in0
	VPXORQ   in0, in0_h, in0
	VMOVDQU8 in0, -64(out)(len*1)
	SUBQ     $64, len
	SUBQ     $64, tmp0
	JG       loop64b
	CMPQ     len, $512
	JGE      aligned
	VZEROUPPER
	RET

one32b:
	VMOVDQU -32(in)(len*1), in0y
	VPSRLQ  $4, in0y, in0_hy
	VPAND   masky, in0_hy, in0_hy
	VPAND   masky, in0y, in0y
	VPSHUFB in0_hy, high_tbly, in0_hy
	VPSHUFB in0y, low_tbly, in0y
	VPXOR   in0y, in0_hy, in0y
	VMOVDQU in0y, -32(out)(len*1)
	SUBQ    $32, len
	CMPQ    len, $0
	JNE     big
	RET

one16b:
	VMOVDQU -16(in)(len*1), in0x
	VPSRLQ  $4, in0x, in0_hx
	VPAND   maskx, in0x, in0x
	VPAND   maskx, in0_hx, in0_hx
	VPSHUFB in0_hx, high_tblx, in0_hx
	VPSHUFB in0x, low_tblx, in0x
	VPXOR   in0x, in0_hx, in0x
	VMOVDQU in0x, -16(out)(len*1)
	SUBQ    $16, len
	CMPQ    len, $0
	JNE     big
	RET

// func coeffMulVectUpdateAVX512(tbl, d, p []byte)
TEXT ·coeffMulVectUpdateAVX512(SB), NOSPLIT, $0
	MOVQ         i+24(FP), in
	MOVQ         o+48(FP), out
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), low_tblx
	VMOVDQU      16(tmp0), high_tblx
	VINSERTI128  $1, low_tblx, low_tbly, low_tbly
	VINSERTI128  $1, high_tblx, high_tbly, high_tbly
	VINSERTI64X4 $1, low_tbly, low_tbl, low_tbl
	VINSERTI64X4 $1, high_tbly, high_tbl, high_tbl
	MOVB         $0x0f, DX
	VPBROADCASTB DX, mask
	MOVQ         in_len+32(FP), len
	TESTQ        $31, len
	JNZ          one16b

big:
	TESTQ $511, len
	JNZ   not_aligned

	// 256bytes/loop
aligned:
	MOVQ $0, pos

loop256b:
	// split low/high part(every byte get 4 low/high bit)
	VMOVDQU8 (in)(pos*1), in0
	VPSRLQ   $4, in0, in0_h
	VPANDQ   mask, in0_h, in0_h
	VPANDQ   mask, in0, in0

	// according low/high part shuffle table, store result in second dst register
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in0, low_tbl, in0
	VPXORQ  in0, in0_h, in0
	VPXORQ  (out)(pos*1), in0, in0

	// store result in memory
	VMOVDQU8 in0, (out)(pos*1)

	VMOVDQU8 64(in)(pos*1), in1
	VPSRLQ   $4, in1, in1_h
	VPANDQ   mask, in1_h, in1_h
	VPANDQ   mask, in1, in1
	VPSHUFB  in1_h, high_tbl, in1_h
	VPSHUFB  in1, low_tbl, in1
	VPXORQ   in1, in1_h, in1
	VPXORQ   64(out)(pos*1), in1, in1
	VMOVDQU8 in1, 64(out)(pos*1)

	VMOVDQU8 128(in)(pos*1), in2
	VPSRLQ   $4, in2, in2_h
	VPANDQ   mask, in2_h, in2_h
	VPANDQ   mask, in2, in2
	VPSHUFB  in2_h, high_tbl, in2_h
	VPSHUFB  in2, low_tbl, in2
	VPXORQ   in2, in2_h, in2
	VPXORQ   128(out)(pos*1), in2, in2
	VMOVDQU8 in2, 128(out)(pos*1)

	VMOVDQU8 192(in)(pos*1), in3
	VPSRLQ   $4, in3, in3_h
	VPANDQ   mask, in3_h, in3_h
	VPANDQ   mask, in3, in3
	VPSHUFB  in3_h, high_tbl, in3_h
	VPSHUFB  in3, low_tbl, in3
	VPXORQ   in3, in3_h, in3
	VPXORQ   192(out)(pos*1), in3, in3
	VMOVDQU8 in3, 192(out)(pos*1)

	VMOVDQU8 256(in)(pos*1), in4
	VPSRLQ   $4, in4, in4_h
	VPANDQ   mask, in4_h, in4_h
	VPANDQ   mask, in4, in4
	VPSHUFB  in4_h, high_tbl, in4_h
	VPSHUFB  in4, low_tbl, in4
	VPXORQ   in4, in4_h, in4
	VPXORQ   256(out)(pos*1), in4, in4
	VMOVDQU8 in4, 256(out)(pos*1)

	VMOVDQU8 320(in)(pos*1), in5
	VPSRLQ   $4, in5, in5_h
	VPANDQ   mask, in5_h, in5_h
	VPANDQ   mask, in5, in5
	VPSHUFB  in5_h, high_tbl, in5_h
	VPSHUFB  in5, low_tbl, in5
	VPXORQ   in5, in5_h, in5
	VPXORQ   320(out)(pos*1), in5, in5
	VMOVDQU8 in5, 320(out)(pos*1)

	VMOVDQU8 384(in)(pos*1), in0
	VPSRLQ   $4, in0, in0_h
	VPANDQ   mask, in0_h, in0_h
	VPANDQ   mask, in0, in0
	VPSHUFB  in0_h, high_tbl, in0_h
	VPSHUFB  in0, low_tbl, in0
	VPXORQ   in0, in0_h, in0
	VPXORQ   384(out)(pos*1), in0, in0
	VMOVDQU8 in0, 384(out)(pos*1)

	VMOVDQU8 448(in)(pos*1), in1
	VPSRLQ   $4, in1, in1_h
	VPANDQ   mask, in1_h, in1_h
	VPANDQ   mask, in1, in1
	VPSHUFB  in1_h, high_tbl, in1_h
	VPSHUFB  in1, low_tbl, in1
	VPXORQ   in1, in1_h, in1
	VPXORQ   448(out)(pos*1), in1, in1
	VMOVDQU8 in1, 448(out)(pos*1)

	ADDQ $512, pos
	CMPQ len, pos
	JNE  loop256b
	VZEROUPPER
	RET

not_aligned:
	TESTQ $63, len
	JNZ   one32b
	MOVQ  len, tmp0
	ANDQ  $511, tmp0

loop64b:
	VMOVDQU8 -64(in)(len*1), in0
	VPSRLQ   $4, in0, in0_h
	VPANDQ   mask, in0_h, in0_h
	VPANDQ   mask, in0, in0
	VPSHUFB  in0_h, high_tbl, in0_h
	VPSHUFB  in0, low_tbl, in0
	VPXORQ   in0, in0_h, in0
	VPXORQ   -64(out)(len*1), in0, in0
	VMOVDQU8 in0, -64(out)(len*1)
	SUBQ     $64, len
	SUBQ     $64, tmp0
	JG       loop64b
	CMPQ     len, $512
	JGE      aligned
	VZEROUPPER
	RET

one32b:
	VMOVDQU -32(in)(len*1), in0y
	VPSRLQ  $4, in0y, in0_hy
	VPAND   masky, in0_hy, in0_hy
	VPAND   masky, in0y, in0y
	VPSHUFB in0_hy, high_tbly, in0_hy
	VPSHUFB in0y, low_tbly, in0y
	VPXOR   in0y, in0_hy, in0y
	VPXOR   -32(out)(len*1), in0y, in0y
	VMOVDQU in0y, -32(out)(len*1)
	SUBQ    $32, len
	CMPQ    len, $0
	JNE     big
	RET

one16b:
	VMOVDQU -16(in)(len*1), in0x
	VPSRLQ  $4, in0x, in0_hx
	VPAND   maskx, in0x, in0x
	VPAND   maskx, in0_hx, in0_hx
	VPSHUFB in0_hx, high_tblx, in0_hx
	VPSHUFB in0x, low_tblx, in0x
	VPXOR   in0x, in0_hx, in0x
	VPXOR   -16(out)(len*1), in0x, in0x
	VMOVDQU in0x, -16(out)(len*1)
	SUBQ    $16, len
	CMPQ    len, $0
	JNE     big
	RET
