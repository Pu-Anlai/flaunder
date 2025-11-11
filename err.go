package main

import "fmt"

type iniParseError struct {
	key     string
	value   string
	section string
}

func (e *iniParseError) Error() string {
	if e.section == "" {
		return fmt.Sprintf("%s: cannot parse key %q (value %q)", e.section, e.key, e.value)
	} else {
		return fmt.Sprintf("cannot parse key %q (value %q)", e.key, e.value)
	}
}

type fileAccessError struct {
	path     string
	fileType string
}

func (e *fileAccessError) Error() string {
	if e.fileType == "" {
		return fmt.Sprintf("cannot access file %q", e.path)
	} else {
		return fmt.Sprintf("cannot access %s file %q", e.fileType, e.path)
	}
}

type dimensionError struct {
	entryName string
}

func (e *dimensionError) Error() string {
	if e.entryName == "" {
		return fmt.Sprintf("global settings do not leave enough space for drawing elements")
	} else {
		return fmt.Sprintf("settings for %q do not leave enough space for drawing all its elements", e.entryName)
	}
}

type fileDecodeError struct {
	path     string
	fileType string
}

func (e *fileDecodeError) Error() string {
	return fmt.Sprintf("cannot decode file %q. valid %s file?", e.path, e.fileType)
}
