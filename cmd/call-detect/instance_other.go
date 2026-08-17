//go:build !windows

package main

func singletonHeld() bool { return false }

func watchRemoteQuit(func()) {}
