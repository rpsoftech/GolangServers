package main

import "log"

// Notifier tells the user about something that needs attention.
type Notifier interface {
	Notify(title, message string)
}

type logNotifier struct{}

func (logNotifier) Notify(title, message string) { log.Printf("NOTICE: %s: %s", title, message) }
