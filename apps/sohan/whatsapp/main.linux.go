//go:build !windows

package main

import "log"

func main() {
	log.Fatal("Only On Windows Please")
}
