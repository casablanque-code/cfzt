//go:build windows

package service

func IsActive(name string) bool { return false }
