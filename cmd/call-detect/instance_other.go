//go:build !windows && !linux && !darwin

package main

func singletonHeld() bool { return false }

func watchRemoteQuit(func()) {}
