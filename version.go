package main

const BuildVersion = "0.1.2"

var CurrentCommit string

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
