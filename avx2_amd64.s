// Reference: www.ssrc.ucsc.edu/Papers/plank-fast13.pdf

#include "textflag.h"

// func mulAVX2(low, high, in, out []byte)
TEXT ·mulAVX2(SB), NOSPLIT, $0
	MOVQ         low+0(FP), AX    // low_table addr
	MOVQ        high+24(FP), BX   // high_table addr
	VMOVDQU      (AX), X0   // low_table
	VMOVDQU      (BX), X1   // high_table
	VINSERTI128  $1, X0, Y0, Y0 // low_table, low_table
	VINSERTI128  $1, X1, Y1, Y1 // high_table, high_table
	MOVQ         in+48(FP), AX  // in_addr
	MOVQ         out+72(FP), BX // out_addr
	WORD         $0x0fb2
	LONG         $0x2069e3c4; WORD $0x00d2
	VPBROADCASTB X2, Y2 // mask
	MOVQ         in_len+56(FP), CX
	SHRQ         $5, CX // num of 32bytes

loop:
	VMOVDQU (AX), Y4 // in_data
	VPSRLQ  $4, Y4, Y5
	VPAND   Y2, Y5, Y6 // high_part of in_data
	VPAND   Y2, Y4, Y7 // low_part of in_data
	VPSHUFB Y6, Y1, Y8  // shuffle high_table
	VPSHUFB Y7, Y0, Y9  // shuffle low_table
	VPXOR   Y8, Y9, Y10 // xor low&high
	VMOVDQU Y10, (BX)
	ADDQ    $32, AX
	ADDQ    $32, BX
	SUBQ    $1, CX
	JNZ     loop
	RET

// func mulXORAVX2(low, high, in, out []byte)
TEXT ·mulXORAVX2(SB), NOSPLIT, $0
	MOVQ         lowTable+0(FP), AX
	MOVQ        high+24(FP), BX
	VMOVDQU      (AX), X0
	VMOVDQU      (BX), X1
	VINSERTI128  $1, X0, Y0, Y0
	VINSERTI128  $1, X1, Y1, Y1
	MOVQ         in+48(FP), AX
	MOVQ         out+72(FP), BX
	WORD         $0x0fb2
	LONG         $0x2069e3c4; WORD $0x00d2
	VPBROADCASTB X2, Y2
	MOVQ         in_len+56(FP), CX
	SHRQ         $5, CX

loop:
	VMOVDQU (AX), Y4
	VMOVDQU (BX), Y11
	VPSRLQ  $4, Y4, Y5
	VPAND   Y2, Y5, Y6
	VPAND   Y2, Y4, Y7
	VPSHUFB Y6, Y1, Y8
	VPSHUFB Y7, Y0, Y9
	VPXOR   Y8, Y9, Y10
	VPXOR   Y10, Y11, Y12 // xor old & new
	VMOVDQU Y12, (BX)
	ADDQ    $32, AX
	ADDQ    $32, BX
	SUBQ    $1, CX
	JNZ     loop
	RET

TEXT ·hasAVX2(SB), NOSPLIT, $0
	XORQ AX, AX
	XORQ CX, CX
	ADDL $7, AX
	CPUID              // when CPUID excutes with AX set to 07H, feature info is ret in BX
	SHRQ $5, BX        // AVX -> BX[5] = 1
	ANDQ $1, BX
	MOVB BX, ret+0(FP)
	RET

