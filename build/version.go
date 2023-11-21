package build

const BuildVersion = "0.1.5"

var CurrentCommit string

func UserVersion() string {
	return BuildVersion + CurrentCommit
}
