package process

import (
	"runtime"
	"testing"
)

func BenchmarkNothing(b *testing.B) {
	for i := 0; i < b.N; i++ {
	}
}

func BenchmarkNumGoroutine(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runtime.NumGoroutine()
	}
}

func BenchmarkStatCPU(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Usage.CPU.cpu(0)
	}
}

func BenchmarkStatRAM(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Usage.RAM.ram()
	}
}
