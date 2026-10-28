//go:build !windows

package main

func newNotifier() Notifier { return logNotifier{} }

func ownsConsole() bool { return false }
