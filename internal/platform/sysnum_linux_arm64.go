//go:build linux && arm64

package platform

// Linux asm-generic syscall table used by arm64: renameat2 = 276.
const sysRenameat2 = 276
