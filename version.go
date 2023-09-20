package main

const BuildVersion = "0.1.4"

var CurrentCommit string

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
