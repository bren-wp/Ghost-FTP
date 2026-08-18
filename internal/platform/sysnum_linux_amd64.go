//go:build linux && amd64

package platform

// Linux arch/x86/entry/syscalls/syscall_64.tbl: renameat2 = 316.
const sysRenameat2 = 316
