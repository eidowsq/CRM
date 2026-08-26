//go:build !windows

package main

func handleWindowsServiceCommand(args []string) (bool, error) { return false, nil }

func shouldRunAsWindowsService() bool { return false }

func runWindowsService() error { return nil }
