// Reference: www.ssrc.ucsc.edu/Papers/plank-fast13.pdf

#include "textflag.h"

#define low_tbl Y0
#define high_tbl Y1
#define in_addr  BX
#define out_addr R9
#define len DI
#define mask Y2

#define in0  Y6
#define in1  Y7
#define in2  Y8
#define in3  Y9
#define in0_h  Y10
#define in1_h  Y11
#define in2_h  Y12
#define in3_h  Y13

#define tmp0 R8

TEXT ·setYMM(SB), NOSPLIT, $0
    MOVQ  a+0(FP), tmp0
    MOVB         $0x0f, DX
    LONG         $0x2069e3c4; WORD $0x00d2   // VPINSRB $0x00, EDX, XMM2, XMM2
   	VPBROADCASTB X2, mask
   	VMOVDQU      mask, (tmp0)
   	RET

TEXT ·returnYMM(SB), NOSPLIT, $0
    MOVQ  a+0(FP), tmp0
    VMOVDQU mask, (tmp0)
    RET

// func vectMulAVX2(tbl, inV, outV []byte)
TEXT ·vectMulAVX2(SB), NOSPLIT, $0
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), X0
	VMOVDQU      16(tmp0), X1
	VINSERTI128  $1, X0, low_tbl, low_tbl
	VINSERTI128  $1, X1, high_tbl, high_tbl
	MOVQ         in+24(FP), in_addr
	MOVQ         out+48(FP), out_addr
	MOVB         $0x0f, DX
	LONG         $0x2069e3c4; WORD $0x00d2   // VPINSRB $0x00, EDX, XMM2, XMM2
	VPBROADCASTB X2, mask
	MOVQ         in_len+32(FP), len
	SHRQ         $7, len

loop:
	VMOVDQU (in_addr), in0
	VMOVDQU 32(in_addr), in1
	VMOVDQU 64(in_addr), in2
	VMOVDQU 96(in_addr), in3

	VPSRLQ  $4, in0, in0_h
	VPSRLQ  $4, in1, in1_h
	VPSRLQ  $4, in2, in2_h
	VPSRLQ  $4, in3, in3_h

	// TODO do I need mask1 2 3 here?
	VPAND   mask, in0_h, in0_h
	VPAND   mask, in1_h, in1_h
	VPAND   mask, in2_h, in2_h
	VPAND   mask, in3_h, in3_h
	VPAND   mask, in0, in0
    VPAND   mask, in1, in1
   	VPAND   mask, in2, in2
   	VPAND   mask, in3, in3

	// TODO do I need tbl1 2 3 here?
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in1_h, high_tbl, in1_h
	VPSHUFB in2_h, high_tbl, in2_h
	VPSHUFB in3_h, high_tbl, in3_h
	VPSHUFB in0, low_tbl, in0
    VPSHUFB in1, low_tbl, in1
   	VPSHUFB in2, low_tbl, in2
   	VPSHUFB in3, low_tbl, in3

	VPXOR   in0, in0_h, in0
	VPXOR   in1, in1_h, in1
	VPXOR   in2, in2_h, in2
	VPXOR   in3, in3_h, in3

	VMOVDQU in0, (out_addr)
	VMOVDQU in1, 32(out_addr)
	VMOVDQU in2, 64(out_addr)
	VMOVDQU in3, 96(out_addr)

	// TODO maybe I can use a POS instead of it
	ADDQ $128, in_addr
	ADDQ $128, out_addr
	SUBQ $1, len
	JG  loop
	
	RET

// func vectMulPlusAVX2(tbl, inV, outV []byte)
TEXT ·vectMulPlusAVX2(SB), NOSPLIT, $0
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), X0
	VMOVDQU      16(tmp0), X1
	VINSERTI128  $1, X0, low_tbl, low_tbl
	VINSERTI128  $1, X1, high_tbl, high_tbl
	MOVQ         in+24(FP), in_addr
	MOVQ         out+48(FP), out_addr
	MOVB         $0x0f, DX
	LONG         $0x2069e3c4; WORD $0x00d2   // VPINSRB $0x00, EDX, XMM2, XMM2
	VPBROADCASTB X2, mask
	MOVQ         in_len+32(FP), len
	SHRQ         $7, len

loop:
	VMOVDQU (in_addr), in0
	VMOVDQU 32(in_addr), in1
	VMOVDQU 64(in_addr), in2
	VMOVDQU 96(in_addr), in3

	VPSRLQ  $4, in0, in0_h
	VPSRLQ  $4, in1, in1_h
	VPSRLQ  $4, in2, in2_h
	VPSRLQ  $4, in3, in3_h

	VPAND   mask, in0_h, in0_h
	VPAND   mask, in1_h, in1_h
	VPAND   mask, in2_h, in2_h
	VPAND   mask, in3_h, in3_h
	VPAND   mask, in0, in0
    VPAND   mask, in1, in1
   	VPAND   mask, in2, in2
   	VPAND   mask, in3, in3

	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in1_h, high_tbl, in1_h
	VPSHUFB in2_h, high_tbl, in2_h
	VPSHUFB in3_h, high_tbl, in3_h
	VPSHUFB in0, low_tbl, in0
    VPSHUFB in1, low_tbl, in1
   	VPSHUFB in2, low_tbl, in2
   	VPSHUFB in3, low_tbl, in3

	VPXOR   in0, in0_h, in0
	VPXOR   in1, in1_h, in1
	VPXOR   in2, in2_h, in2
	VPXOR   in3, in3_h, in3

	VPXOR   (out_addr), in0, in0
    VPXOR   32(out_addr), in1, in1
   	VPXOR   64(out_addr), in2, in2
   	VPXOR   96(out_addr), in3, in3

	VMOVDQU in0, (out_addr)
	VMOVDQU in1, 32(out_addr)
	VMOVDQU in2, 64(out_addr)
	VMOVDQU in3, 96(out_addr)

	ADDQ $128, in_addr
	ADDQ $128, out_addr
	SUBQ $1, len
	JG  loop
	
	RET

// func vectMulAVX2_32B(tbl, inV, outV []byte)
TEXT ·vectMulAVX2_32B(SB), NOSPLIT, $0
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), X0
	VMOVDQU      16(tmp0), X1
	VINSERTI128  $1, X0, low_tbl, low_tbl
	VINSERTI128  $1, X1, high_tbl, high_tbl
	MOVQ         in+24(FP), in_addr
	MOVQ         out+48(FP), out_addr
	MOVB         $0x0f, DX
	LONG         $0x2069e3c4; WORD $0x00d2   // VPINSRB $0x00, EDX, XMM2, XMM2
	VPBROADCASTB X2, mask
	MOVQ         in_len+32(FP), len
	SHRQ         $5, len

loop:
	VMOVDQU (in_addr), in0
	VPSRLQ  $4, in0, in0_h
	VPAND   mask, in0_h, in0_h
	VPAND   mask, in0, in0
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in0, low_tbl, in0
	VPXOR   in0, in0_h, in0
	VMOVDQU in0, (out_addr)
	ADDQ $32, in_addr
	ADDQ $32, out_addr
	SUBQ $1, len
	JG  loop
	
	RET

// func vectMulPlusAVX2_32B(tbl, inV, outV []byte)
TEXT ·vectMulPlusAVX2_32B(SB), NOSPLIT, $0
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), X0
	VMOVDQU      16(tmp0), X1
	VINSERTI128  $1, X0, low_tbl, low_tbl
	VINSERTI128  $1, X1, high_tbl, high_tbl
	MOVQ         in+24(FP), in_addr
	MOVQ         out+48(FP), out_addr
	MOVB         $0x0f, DX
	LONG         $0x2069e3c4; WORD $0x00d2   // VPINSRB $0x00, EDX, XMM2, XMM2
	VPBROADCASTB X2, mask
	MOVQ         in_len+32(FP), len
	SHRQ         $5, len

loop:
	VMOVDQU (in_addr), in0
	VPSRLQ  $4, in0, in0_h
	VPAND   mask, in0_h, in0_h
	VPAND   mask, in0, in0
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in0, low_tbl, in0
	VPXOR   in0, in0_h, in0
	VPXOR   (out_addr), in0, in0
	VMOVDQU in0, (out_addr)
	ADDQ $32, in_addr
	ADDQ $32, out_addr
	SUBQ $1, len
	JG  loop
	
	RET

TEXT ·hasAVX2(SB), NOSPLIT, $0
	XORQ AX, AX
	XORQ CX, CX
	ADDL $7, AX
	CPUID
	SHRQ $5, BX
	ANDQ $1, BX
	MOVB BX, ret+0(FP)
	RET
