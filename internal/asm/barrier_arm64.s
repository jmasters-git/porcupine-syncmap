//go:build arm64

#include "textflag.h"

TEXT ·MemoryBarrier(SB), NOSPLIT|NOFRAME, $0-0
	DMB $0xb // DMB ISH
	RET
