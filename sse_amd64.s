// Reference: www.ssrc.ucsc.edu/Papers/plank-fast13.pdf

#include "textflag.h"

#define low_tbl X0
#define high_tbl X1
#define in_addr  BX
#define out_addr SI
#define len DI
#define mask X2

#define in0  X6
#define in1  X7
#define in0_h  X10
#define in1_h  X11

#define tmp0 R8
#define xtmp0 X3
#define xtmp1 X4
#define xtmp2 X5
#define xtmp3 X8

// func copy32B(dst, src []byte)
TEXT ·copy32B(SB), NOSPLIT, $0
    MOVQ dst+0(FP), AX
    MOVQ src+24(FP), BX
    MOVOU (BX), X0
    MOVOU 16(BX), X1
    MOVOU X0, (AX)
    MOVOU X1, 16(AX)
    VZEROUPPER
    RET

// func vectMulSSSE3(tbl, inV, outV []byte)
TEXT ·vectMulSSSE3(SB), NOSPLIT, $0
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), low_tbl
	VMOVDQU      16(tmp0), high_tbl
	MOVQ         in+24(FP), in_addr
	MOVQ         out+48(FP), out_addr
	XORQ   tmp0, tmp0
    MOVB   $15, tmp0
    MOVQ   tmp0, mask
    PXOR   xtmp0, xtmp0
   	PSHUFB xtmp0, mask
	MOVQ         in_len+32(FP), len
	SHRQ         $5, len

loop:
	MOVOU  (in_addr), in0
	MOVOU 16(in_addr), in1

	MOVOU in0, in0_h
	MOVOU in1, in1_h

	PSRLQ $4, in0_h
	PSRLQ $4, in1_h

	PAND  mask, in0
	PAND  mask, in1
	PAND  mask, in0_h
	PAND  mask, in1_h

	MOVOU low_tbl, xtmp0
	MOVOU low_tbl, xtmp1
	MOVOU high_tbl, xtmp2
	MOVOU high_tbl, xtmp3

	PSHUFB in0, xtmp0
	PSHUFB in1, xtmp1
	PSHUFB in0_h, xtmp2
	PSHUFB in1_h, xtmp3

	PXOR   xtmp0, xtmp2
	PXOR   xtmp1, xtmp3
	MOVOU  xtmp2, (out_addr)
	MOVOU  xtmp3, 16(out_addr)

	ADDQ $32, in_addr
	ADDQ $32, out_addr
	SUBQ $1, len
	JG  loop
	VZEROUPPER
	RET

// func vectMulPlusSSSE3(tbl, inV, outV []byte)
TEXT ·vectMulPlusSSSE3(SB), NOSPLIT, $0
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), low_tbl
	VMOVDQU      16(tmp0), high_tbl
	MOVQ         in+24(FP), in_addr
	MOVQ         out+48(FP), out_addr
	XORQ   tmp0, tmp0
    MOVB   $15, tmp0
    MOVQ   tmp0, mask
    PXOR   xtmp0, xtmp0
   	PSHUFB xtmp0, mask
	MOVQ         in_len+32(FP), len
	SHRQ         $5, len

loop:
	MOVOU  (in_addr), in0
	MOVOU 16(in_addr), in1

	MOVOU in0, in0_h
	MOVOU in1, in1_h

	PSRLQ $4, in0_h
	PSRLQ $4, in1_h

	PAND  mask, in0
	PAND  mask, in1
	PAND  mask, in0_h
	PAND  mask, in1_h

	MOVOU low_tbl, xtmp0
	MOVOU low_tbl, xtmp1
	MOVOU high_tbl, xtmp2
	MOVOU high_tbl, xtmp3

	PSHUFB in0, xtmp0
	PSHUFB in1, xtmp1
	PSHUFB in0_h, xtmp2
	PSHUFB in1_h, xtmp3

	PXOR   xtmp0, xtmp2
	PXOR   xtmp1, xtmp3
	XORPD (out_addr), xtmp2
	XORPD 16(out_addr), xtmp3
	MOVOU  xtmp2, (out_addr)
	MOVOU  xtmp3, 16(out_addr)

	ADDQ $32, in_addr
	ADDQ $32, out_addr
	SUBQ $1, len
	JG  loop
	VZEROUPPER
	RET

// func vectMulSSSE3_16B(tbl, inV, outV []byte)
TEXT ·vectMulSSSE3_16B(SB), NOSPLIT, $0
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), low_tbl
	VMOVDQU      16(tmp0), high_tbl
	MOVQ         in+24(FP), in_addr
	MOVQ         out+48(FP), out_addr
	XORQ   tmp0, tmp0
    MOVB   $15, tmp0
    MOVQ   tmp0, mask
    PXOR   xtmp0, xtmp0
   	PSHUFB xtmp0, mask
	MOVQ         in_len+32(FP), len
	SHRQ         $4, len

loop:
	MOVOU  (in_addr), in0
	MOVOU in0, in0_h
	PSRLQ $4, in0_h
	PAND  mask, in0
	PAND  mask, in0_h
	MOVOU low_tbl, xtmp0
	MOVOU high_tbl, xtmp2
	PSHUFB in0, xtmp0
	PSHUFB in0_h, xtmp2
	PXOR   xtmp0, xtmp2
	MOVOU  xtmp2, (out_addr)
	ADDQ $16, in_addr
	ADDQ $16, out_addr
	SUBQ $1, len
	JG  loop
	VZEROUPPER
	RET

// func vectMulPlusSSSE3_16B(tbl, inV, outV []byte)
TEXT ·vectMulPlusSSSE3_16B(SB), NOSPLIT, $0
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), low_tbl
	VMOVDQU      16(tmp0), high_tbl
	MOVQ         in+24(FP), in_addr
	MOVQ         out+48(FP), out_addr
	XORQ   tmp0, tmp0
    MOVB   $15, tmp0
    MOVQ   tmp0, mask
    PXOR   xtmp0, xtmp0
   	PSHUFB xtmp0, mask
	MOVQ         in_len+32(FP), len
	SHRQ         $4, len

loop:
	MOVOU  (in_addr), in0
	MOVOU in0, in0_h
	PSRLQ $4, in0_h
	PAND  mask, in0
	PAND  mask, in0_h
	MOVOU low_tbl, xtmp0
	MOVOU high_tbl, xtmp2
	PSHUFB in0, xtmp0
	PSHUFB in0_h, xtmp2
	PXOR   xtmp0, xtmp2
	XORPD (out_addr), xtmp2
	MOVOU  xtmp2, (out_addr)
	ADDQ $16, in_addr
	ADDQ $16, out_addr
	SUBQ $1, len
	JG  loop
	VZEROUPPER
	RET

TEXT ·hasSSSE3(SB), NOSPLIT, $0
	XORQ AX, AX
	INCL AX
	CPUID
	SHRQ $9, CX
	ANDQ $1, CX
	MOVB CX, ret+0(FP)
	RET
