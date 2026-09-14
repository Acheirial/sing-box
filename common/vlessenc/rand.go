package vlessenc

import (
	"crypto/rand"
	"math/big"
	"runtime"

	"golang.org/x/sys/cpu"
)

// RandBetween returns a uniformly random int64 in [from, to), ported
// verbatim from Xray-core common/crypto.RandBetween (inclusive lower
// bound, exclusive upper bound; argument order irrelevant; a range of
// 0 or 1 returns from).
func RandBetween(from int64, to int64) int64 {
	if from > to {
		from, to = to, from
	}
	if d := to - from; d == 0 || d == 1 {
		return from
	}
	bigInt, _ := rand.Int(rand.Reader, big.NewInt(to-from))
	return from + bigInt.Int64()
}

// hasGCMAsmAMD64 and friends mirror Xray-core common/protocol/headers.go,
// kept in sync with crypto/tls/cipher_suites.go: the AEAD choice between
// AES-128-GCM and ChaCha20-Poly1305 must match the peer's TLS stack so
// the encrypted outer records blend with the surrounding TLS traffic.
var (
	hasGCMAsmAMD64 = cpu.X86.HasAES && cpu.X86.HasPCLMULQDQ && cpu.X86.HasSSE41 && cpu.X86.HasSSSE3
	hasGCMAsmARM64 = (cpu.ARM64.HasAES && cpu.ARM64.HasPMULL) || (runtime.GOOS == "darwin" && runtime.GOARCH == "arm64")
	hasGCMAsmS390X = cpu.S390X.HasAES && cpu.S390X.HasAESCTR && cpu.S390X.HasGHASH
	hasGCMAsmPPC64 = runtime.GOARCH == "ppc64" || runtime.GOARCH == "ppc64le"
)

var HasAESGCMHardwareSupport = hasGCMAsmAMD64 || hasGCMAsmARM64 || hasGCMAsmS390X || hasGCMAsmPPC64
