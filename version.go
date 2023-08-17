package main

const BuildVersion = "0.1.0"

var CurrentCommit string

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
