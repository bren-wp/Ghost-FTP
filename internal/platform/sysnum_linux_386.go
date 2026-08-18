//go:build linux && 386

package platform

// Linux arch/x86/entry/syscalls/syscall_32.tbl: renameat2 = 353.
const sysRenameat2 = 353
