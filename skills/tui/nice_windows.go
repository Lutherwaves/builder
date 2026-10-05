package main

import "syscall"

// lowestPriority runs the cockpit, and every command it starts, in the idle
// priority class.
func lowestPriority() {
	const idlePriorityClass = 0x40
	k := syscall.NewLazyDLL("kernel32.dll")
	self, _, _ := k.NewProc("GetCurrentProcess").Call()
	_, _, _ = k.NewProc("SetPriorityClass").Call(self, idlePriorityClass)
}
