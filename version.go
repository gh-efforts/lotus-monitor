package main

const BuildVersion = "0.1.3"

var CurrentCommit string

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
