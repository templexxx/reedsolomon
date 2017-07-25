#include "textflag.h"

// func mulSSSE3(low, high, in, out []byte)
TEXT ·mulSSSE3(SB), NOSPLIT, $0
	MOVQ   low+0(FP), AX
	MOVQ   high+24(FP), BX
	MOVOU  (AX), X0
	MOVOU  (BX), X1
	MOVQ   in+48(FP), AX
	MOVQ   out+72(FP), BX
	MOVQ   $15, CX
	MOVQ   CX, X3
	PXOR   X2, X2
	PSHUFB X2, X3
	MOVQ   in_len+56(FP), CX
	SHRQ   $4, CX

loop:
	MOVOU  (AX), X4
	MOVOU  X4, X5
	PSRLQ  $4, X5
	PAND   X3, X4
	PAND   X3, X5
	MOVOU  X0, X6
	MOVOU  X1, X7
	PSHUFB X4, X6
	PSHUFB X5, X7
	PXOR   X6, X7
	MOVOU  X7, (BX)
	ADDQ   $16, AX
	ADDQ   $16, BX
	SUBQ   $1, CX
	JNZ    loop
	RET

// func mulXORSSSE3(low, high, in, out []byte)
TEXT ·mulXORSSSE3(SB), NOSPLIT, $0
	MOVQ   low+0(FP), AX
	MOVQ   high+24(FP), BX
	MOVOU  (AX), X0
	MOVOU  (BX), X1
	MOVQ   in+48(FP), AX
	MOVQ   out+72(FP), BX
	MOVQ   $15, CX
	MOVQ   CX, X3
	PXOR   X2, X2
	PSHUFB X2, X3
	MOVQ   in_len+56(FP), CX
	SHRQ   $4, CX

loop:
	MOVOU  (AX), X4
	MOVOU  (BX), X8
	MOVOU  X4, X5
	PSRLQ  $4, X5
	PAND   X3, X4
	PAND   X3, X5
	MOVOU  X0, X6
	MOVOU  X1, X7
	PSHUFB X4, X6
	PSHUFB X5, X7
	PXOR   X6, X7
	PXOR   X8, X7
	MOVOU  X7, (BX)
	ADDQ   $16, AX
	ADDQ   $16, BX
	SUBQ   $1, CX
	JNZ    loop
	RET

TEXT ·hasSSSE3(SB), NOSPLIT, $0
	XORQ AX, AX
	INCL AX
	CPUID              // when CPUID excutes with AX set to 01H, feature info is ret in CX and DX
	SHRQ $9, CX        // SSSE3 -> CX[9] = 1
	ANDQ $1, CX
	MOVB CX, ret+0(FP)
	RET
