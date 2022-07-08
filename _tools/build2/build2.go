/*
build2.go is a rewrite of build.go using some ideas from the restic build script.

THIS DOES NOT WORK AS OF NOW!!!!!!
*/

package main

import (
	"flag"
	"os"
	"path/filepath"
)

//binary building notes
const (
	//repo is the path off $GOPATH/src to the repo storing the code to be built. If
	//empty, the current directory is used.
	repo = "github.com/c9845/licensekeys"

	//binaryName is what the binary will be named when built, exe added for windows.
	binaryName = "licensekeys"

	//buildsDir is the directory to saves built binaries to. If empty, the current
	//directory is used.
	buildsDir = "_builds"

	//ldFlags lists the default ldflags passed to go build.
	ldFlags = "-s -w"
)

//osArch defines the types of distributions we can build for. These match GOOS, GOARCH,
//and GOARM values.
type osArch struct {
	OS   string //windows, linux, darwin (mac os)
	Arch string //amd64, arm64
	Arm  string //6, 7, blank for latest (use 6 to support older raspberry pi)
}

//definedOSArchs is the list of types of binaries that will be built.
var definedOSArchs = []osArch{
	{
		OS:   "windows",
		Arch: "amd64",
	},
	{
		OS:   "linux", //linux with intel/amd
		Arch: "amd64",
	},
	{
		OS:   "darwin", //mac with intel
		Arch: "amd64",
	},
}

func main() {
	//Handle flags.
	binaryName := flag.String("o", binaryName, "Set output binary name.")
	tags := flag.String("tags", "", "Tags passed to 'go build -tags'.")
	goos := flag.String("goos", "", "What OS to build for, if blank, multiple OSes are built for.")
	goarch := flag.String("goarch", "", "What arch to build for, if blank, multiple arches are built for.")
	goarm := flag.String("goarm", "", "What arm level to build for, if blank, 8 is used.")
	cgo := flag.Bool("cgo", false, "Enable CGO.")
	flag.Parse()

	//Get absolute path to repo containing package to build. This path should contain
	//a file with "package main".
	if repo == "" {
		path, err := os.Getwd()
		repo = path
	} else {
		//see if path provided exists
		_, err := os.Stat(repo)
		if os.IsNotExist(err) {
			//repo path provided not found, prepend gopath
			repo = filepath.Join(os.Getenv("GOPATH"), "src", repo)

			//see if path provided exists not that gopath was prepended
			if _, err := os.Stat(err); err != nil {

			}
		}

	}

}
