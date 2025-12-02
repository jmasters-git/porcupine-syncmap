//go:build amd64

#include "textflag.h"

TEXT ·MemoryBarrier(SB), NOSPLIT|NOFRAME, $0-0
	MFENCE
	RET
