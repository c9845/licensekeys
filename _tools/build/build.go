/*
Build.go builds software releases. This builds the binary and gathers other files
needed for running the binary (html, css, js source files, documentation, etc.). The
binary and files are then bundled into zip files for easy distribution.

NOTES:
	- Including HTML, CSS, and JS files isn't really needed since they will be embedded
	  into the binary and using embedded files is nicer (less separate files to
	  distribute).
	- You need to set path to Pandoc if you want to convert files from Markdown to
	  plain text or PDF.

CGO NOTE:
	- This uses a build tag to use the modernc SQLite library for the built binaries
	  so that CGO can be disabled/turned off and we can build a statically linked
	  binary with no GCC dependency. If you use the mattn SQLite library, the built
	  binaries will not be statically linked and may not work on every system you
	  are building for.
*/
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//osArch stores an OS and CPU architecture pairs. See definedOSArchs.
type osArch struct {
	OS   string //windows, linux, darwin
	Arch string //usually amd64
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
	// {
	// 	OS:   "linux", //linus with arm
	// 	Arch: "arm64",
	// },
	{
		OS:   "darwin", //mac with intel
		Arch: "amd64",
	},
	// {
	// 	OS:   "darwin", //mac with arm/apple
	// 	Arch: "arm64",
	// },
}

//execution related
const (
	//gopathRepoSubdirectory is the path to the repo as a subdirectory to GOPATH.
	//This is used to build the complete path to the repo after determining what
	//the GOPATH environmental variable is set to.
	//
	//This is needed since we need to know the directory in wich the to-be-built
	//files are located. We cannot just base this off of the "build.go" directory
	//since when using "go run build.go" the build executable's path is just a
	//temporary directory. So, therefore, we just base the repo directory off of
	//GOPATH since any system using go should have this path set and we assume
	//this repo is located off of the GOPATH.
	//TODO: determine path not using GOPATH.
	gopathRepoSubdirectory = "github.com/c9845/licensekeys"

	//appName is the name of binary output from "go build".
	appName = "licensekeyserver"

	//buildsDirectoryName is the name of the directory off of gopathRepoSubdirectory
	//where the zipped releases, containing the binary and other files, will be
	//saved.
	buildsDirectoryName = "_builds"

	//pathToPandoc is the path the the pandoc executable that converts markdown
	//files to plain text files. Converting files to plain text is useful since
	//almost all systems will present a default application for opening .txt files.
	//.md files will most likely require an app to be chosen to open the files and
	//the files will be ugly (with markdown special chars). If this is blank, no
	//conversion will be done.
	pathToPandoc = ""

	//skipZip prevents zip files from being created. Used for diagnostics.
	skipZip = true

	//enabled cgo
	cgoEnabled = false
)

//directories, files, etc.
var (
	//commonDirectories is the list of directories storing files we want to copy
	//from source code to be distributed with the built binary. These directories
	//typically include website/gui files (html, css, js) even though the binary
	//may embed them as well, and documentation regarding install, updates, and
	//help.
	commonDirectories = []string{
		"website",
		"_documentation",
	}

	//commonRootDirFiles is the list of files stored at the root of the repo that
	//we want to copy from source code to be distributed with the build binary.
	commonRootDirFiles = []string{
		"README.md",
		"COPYRIGHT.md",
		"LICENSE.md",
	}

	//nonPublicCommonFilesPaths is a list of paths, based of the common temporary
	//directory used when building releases, of files that should not be included
	//in the release. These are typically files located within a commonDirectory
	//but we do not want to distribute. Typically this is used for removing non-
	//minified js and css files as well as test or development images, docs, etc.
	//
	//Note that the full path the the temporary common files directory will be
	//prepended to each path provided. The paths provide here are just partial
	//paths.
	nonPublicCommonFilesPaths = []string{
		filepath.Join("website", "static", "css", "styles.css"),
		filepath.Join("website", "static", "js", "script.js"),
	}

	//nonPublicDirectories is a list of paths, based off the common temporary
	//directory used when building releases, of directories that should be be
	//incldued in the release. These are typically directories that are used for
	//storing development or test files.
	//
	//Note that the full path the the temporary common files directory will be
	//prepended to each path provided. The paths provide here are just partial
	//paths.
	nonPublicDirectories = []string{
		filepath.Join("_documentation", "not_public"),
		filepath.Join("website", "static", "images", "not_public"),
	}
)

func main() {
	//Determine path to repo.
	pathToRepo, err := getRepoPath()
	if err != nil {
		log.Fatalln("Could not get path to repo.", err)
		return
	}
	log.Println("Repo Path:", pathToRepo)

	//Create builds output directory. This is where the builds will be stored after
	//being completely built and zipped.
	pathToBuilds := filepath.Join(pathToRepo, buildsDirectoryName)
	err = os.MkdirAll(pathToBuilds, 0777)
	if err != nil {
		log.Fatalln("Could not create builds directory.", err)
		return
	}
	log.Println("Builds Path:", pathToBuilds)

	//Get main app version from changelog. This will be used in names of built zip
	//files and compared against version number returned from binaries.
	changelogVersion, err := getChangelogVersion(pathToRepo)
	if err != nil {
		log.Fatalln("Could not get version from changelog.", err)
		return
	}
	log.Println("Changelog Version:", changelogVersion)

	//Create directory where all common files used in builds for various os/arch are
	//stored. This is done so that we can just copy one directory for each os/arch
	//instead of having to perform the copying of each common item over and over.
	//We make sure the directory doesn't already exist to handle errors with
	//overwriting already existing files or directories.
	pathToCommonFiles, err := createCommonDirectory(changelogVersion, pathToBuilds)
	if err != nil {
		log.Fatalln("Error creating common dir", err)
		return
	}
	log.Println("Common Files Dir:", pathToCommonFiles)

	//Copy common directories. These are directories at the "root" for the app's
	//repo. Ex.: website/, _documentation/.
	err = copyCommonDirectories(pathToRepo, pathToCommonFiles)
	if err != nil {
		log.Fatalln("Copy common directories error", err)
		return
	}

	//Copy common files. These are files at the "root" of the app's repo.
	//Ex.: README.md, COPYRIGHT.md.
	err = copyCommonRootFiles(pathToRepo, pathToCommonFiles)
	if err != nil {
		log.Fatalln("Copy root files error", err)
		return
	}

	//Remove source files and non-minified files from copied files. These files are
	//not needed or not wanted in built releases.
	//Ex: styles.css, script.js.
	err = removeNonPublicFiles(pathToCommonFiles)
	if err != nil {
		log.Fatalln("Error removing non public files", err)
		return
	}

	//Convert markdown files to plain text files. This only affects files copied
	//to the common-files-for-builds directory so we don't modify any source files
	//by mistake. Conversion is skipped if a path to pandoc isn't provided.
	err = convertMarkdownToText(pathToCommonFiles)
	if err != nil {
		log.Fatalln("Error converting files", err)
		return
	}

	//Build the binaries.
	err = buildBinaries(pathToRepo, pathToBuilds, changelogVersion, pathToCommonFiles, definedOSArchs)
	if err != nil {
		log.Fatalln("Error building", err)
		return
	}

	//Remove common directory. This is just a list of copied files so we can get
	//rid of it without concern.
	err = os.RemoveAll(pathToCommonFiles)
	if err != nil {
		log.Fatalln("Could not remove common directory", err)
		return
	}

	//done
}

//getRepoPath gets the full, absolute path to repo.
//
//This assumes the repo is located off of GOPATH, but basing a repo off GOPATH
//is an old assumption since GOMOD allows repos to be located anywhere. Need to
//find a way to get path to repo outside of GOPATH.
//TODO: determine path not using GOPATH.
func getRepoPath() (path string, err error) {
	gopath := os.Getenv("GOPATH")
	path, err = filepath.Abs(filepath.Join(gopath, "src", gopathRepoSubdirectory))
	return
}

//getChangelogVersion reads the version from the changelog file. This version will
//be used to construct version-numbered release zip files and also check that the
//app version (via binary --version flag) matches. Note that the returned value will
//be in the format "v1.2.3" with the prepended "v" coming from the changelog file.
func getChangelogVersion(repoPath string) (version string, err error) {
	//build path to changelog file
	pathToChangelogFile := filepath.Join(repoPath, "_documentation", "changelog.txt")

	//open the changelog file
	f, err := os.Open(pathToChangelogFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	//read the first line of the changelog file which should be the latest version
	//number.
	b := bufio.NewReader(f)
	firstLine, err := b.ReadSlice('\n')
	if err != nil {
		return "", err
	}

	version = strings.TrimSpace(string(firstLine))
	return version, nil
}

//createCommonDirectory creates the ./_builds/common-v1.1.1 (or whatever version is
//applicable) where the common files between os/arch builds are stored.
func createCommonDirectory(version, buildsPath string) (commonPath string, err error) {
	//create directory full name
	commonDirectoryName := "common-" + version

	//get full path to common directory
	commonPath = filepath.Join(buildsPath, commonDirectoryName)

	//create the directory
	//Make sure this directory doesn't already exist. If it does, delete
	//it since we want to recopy all files in case any have changed.
	_, err = os.Stat(commonPath)
	if err != nil && !os.IsNotExist(err) {
		//unknown error
		return "", err

	} else if err == nil {
		//directory already exists, remove it
		err := os.RemoveAll(commonPath)
		if err != nil {
			return "", err
		}
	}

	//create the directory
	err = os.MkdirAll(commonPath, 0777)
	if err != nil {
		return "", err
	}

	return commonPath, nil
}

//copyCommonDirectories copies directories that contain common files used for
//all arch/os builds. These directories include files you want to distribute with
//the built binary. Typically html, css, js files and intall, updates, and other
//documentation.
func copyCommonDirectories(repoPath, commonDirPath string) error {
	for _, d := range commonDirectories {
		src := filepath.Join(repoPath, d)
		dst := filepath.Join(commonDirPath, d)
		err := copyDir(src, dst)
		if err != nil {
			return err
		}
	}

	return nil
}

//copyCommonRootFiles copies files that are stored at the root of repo directory
//to be distributed with the built binary. Typically used for README and COPYRIGHT
//type files.
func copyCommonRootFiles(repoPath, commonDirPath string) error {
	for _, f := range commonRootDirFiles {
		src := filepath.Join(repoPath, f)
		dst := filepath.Join(commonDirPath, f)
		err := copyFile(src, dst)
		if err != nil {
			return err
		}
	}

	return nil
}

//removeNonPublicFiles removes files from the common directory that are not
//for end users.  These are non-minified js and css source files as well as
//not public documentation files.
func removeNonPublicFiles(commonDirPath string) error {
	//Remove each specified non-public file from the common files.
	for _, f := range nonPublicCommonFilesPaths {
		//prepend the full path to the common directory to the partial path to
		//each file.
		f = filepath.Join(commonDirPath, f)

		err := os.Remove(f)
		if err != nil {
			return err
		}
	}

	//Remove typescript source files.
	jsFilesDir := filepath.Join(commonDirPath, "website", "static", "js")
	jsFiles, err := os.ReadDir(jsFilesDir)
	if err != nil {
		return err
	}
	for _, f := range jsFiles {
		if f.IsDir() {
			continue
		}

		if filepath.Ext(f.Name()) == ".ts" {
			path := filepath.Join(jsFilesDir, f.Name())
			err := os.Remove(path)
			if err != nil {
				return err
			}
		}
	}

	//Remove other non-public directories.
	for _, path := range nonPublicDirectories {
		//prepend the full path to the common directory to the partial path to
		//each directory
		path = filepath.Join(commonDirPath, path)

		err := os.RemoveAll(path)
		if err != nil {
			return err
		}
	}

	return nil
}

//convertMarkdownToText converts md files to txt using pandoc.
//This works through subdirectories searching for files ending in .md and converting
//then to txt files. We do this since txt files are easier to open on end-user
//systems. This removes the .md files from the common directory so they aren't
//distributed.
func convertMarkdownToText(pathToParentDirectory string) error {
	if pathToPandoc == "" {
		log.Println("Converstion from Markdown to TXT skipped, no Pandoc path provided.")
		return nil
	}

	//read through files in directory.
	files, err := os.ReadDir(pathToParentDirectory)
	if err != nil {
		return err
	}
	for _, f := range files {
		fullFilePath := filepath.Join(pathToParentDirectory, f.Name())

		//if we find a directory, navigate into it to check for .md files.
		if f.IsDir() {
			convertMarkdownToText(fullFilePath)
			continue
		}

		//found .md file, convert to txt and remove .md file.
		if filepath.Ext(fullFilePath) == ".md" {
			// log.Println("Converting file:", fullFilePath)

			outputFilePath := strings.Replace(fullFilePath, ".md", ".txt", 1)
			args := []string{
				"-f", "markdown",
				"-t", "plain", //.txt
				fullFilePath,
				"-o", outputFilePath,
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd := exec.Command(pathToPandoc, args...)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if err != nil {
				log.Println("MD to TXT error (stdout):", stdout.String())
				log.Println("MD to TXT error (stderr):", stderr.String())
				return err
			}

			// log.Println("Converting file:", fullFilePath, "...done")

			//remove .md file
			err = os.Remove(fullFilePath)
			if err != nil {
				return err
			}
		} //end if: found .md file
	} //end for: loop through files in directory

	return nil
}

//buildBinaries creates the executable/binarie for each os/arch pairing, saves it
//to an output directory for that build, copies over the common files, compresses
//the directory storing the binary and other files, and checksums the built zip file.
func buildBinaries(repoPath, buildsDirPath, changelogVersion, commonDirPath string, osArches []osArch) error {
	//loop through each os/arch pairing
	for _, osarch := range definedOSArchs {
		log.Println("Building...", osarch.OS, osarch.Arch)

		//set GO environmental vars
		err := os.Setenv("GOOS", osarch.OS)
		if err != nil {
			return err
		}
		err = os.Setenv("GOARCH", osarch.Arch)
		if err != nil {
			return err
		}

		//create name for binary
		//add .exe extension for windows builds.
		binaryName := appName
		if osarch.OS == "windows" {
			binaryName = appName + ".exe"
		}

		//create directory for this specific build
		dir := appName + "-" + changelogVersion + "-" + osarch.OS + "_" + osarch.Arch
		pathToBinaryDir := filepath.Join(buildsDirPath, dir)
		createBuildOutputDirectory(pathToBinaryDir)

		//get path to go to call "go build" with.
		pathToGo := ""
		switch runtime.GOOS {
		case "windows":
			pathToGo = filepath.Join("C:/", "Program Files", "go", "bin", "go.exe")
		case "linux":
			pathToGo = filepath.Join("/", "usr", "local", "go", "bin", "go")
		case "darwin":
			pathToGo = filepath.Join("/", "usr", "local", "go", "bin", "go")
		default:
			return errors.New("unhandled GOOS " + runtime.GOOS)
		}

		//create the binary
		pathToBinary := filepath.Join(pathToBinaryDir, binaryName)
		args := []string{
			//go build...
			"build",

			//remove absolute system paths from panic/trace outputs
			//only relative parts of paths are shown
			//end users don't need to see full paths of developer's machines.
			"-trimpath",

			//strip some debugging info to make binary smaller
			"-ldflags", "-s -w",

			//use modernc sqlite library since this allows for not using CGO and for
			//building staticly linked builds.
			"-tags", "modernc",

			//set binary output name
			//using "-o " + pathToBinary doesn't work for some reason.
			"-o", pathToBinary,
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd := exec.Command(pathToGo, args...)
		cmd.Dir = repoPath
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if !cgoEnabled {
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "CGO_ENABLED=0") //never used cgo
		}

		err = cmd.Run()
		if err != nil {
			log.Println("Build error (stdout):", stdout.String())
			log.Println("Build error (stderr):", stderr.String())
			log.Println("Cmd + args:          ", pathToGo, args)
			return err
		}

		//TODO: save hash of binary? so that binary can be checked for modification, not just zip file?

		//copy common files to build directory
		files, err := os.ReadDir(commonDirPath)
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.IsDir() {
				src := filepath.Join(commonDirPath, f.Name())
				dst := filepath.Join(pathToBinaryDir, f.Name())
				err := copyDir(src, dst)
				if err != nil {
					return err
				}
			} else {
				src := filepath.Join(commonDirPath, f.Name())
				dst := filepath.Join(pathToBinaryDir, f.Name())
				err := copyFile(src, dst)
				if err != nil {
					return err
				}
			}
		}

		//make sure zipped file for this build doesn't already exist
		//same for hash file
		zipFilePath := filepath.Join(pathToBinaryDir + ".zip")
		err = os.Remove(zipFilePath)
		if os.IsNotExist(err) == false && err != nil {
			return err
		}
		hashFilePath := zipFilePath + ".sha256"
		err = os.Remove(hashFilePath)
		if os.IsNotExist(err) == false && err != nil {
			return err
		}

		//skipping zipping if needed, mostly for diagnostics
		if skipZip {
			log.Println("Skipping zip file creation and hashing.")
			continue
		}

		//create zip file of build directory
		//create the zip output file (the compressed file)
		zipFile, err := os.Create(zipFilePath)
		if err != nil {
			return err
		}

		//get writer to use for creating zip and create zip
		//Make sure writer is closed upon errors or after creating
		//zip successfully.
		zw := zip.NewWriter(zipFile)
		defer zw.Close()

		createZip(zw, pathToBinaryDir, "")
		zw.Close()

		//remove build directory
		err = os.RemoveAll(pathToBinaryDir)
		if err != nil {
			return err
		}

		//create hash of zip file and save to file
		z, err := os.Open(zipFilePath)
		if err != nil {
			return err
		}
		defer z.Close()

		h := sha256.New()
		_, err = io.Copy(h, z)
		if err != nil {
			return err
		}

		hash := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
		err = os.WriteFile(hashFilePath, []byte(hash), os.ModeAppend)
		if err != nil {
			return err
		}
	} //end for: loop through os/arches

	return nil
}

//createBuildOutputDirectory creates directory where a build binary for an os/arch
//is stored
func createBuildOutputDirectory(path string) error {
	//create the directory
	//Make sure this directory doesn't already exist. If it does, delete
	//it since we want to recopy all files in case any have changed.
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) == false {
		return err

	} else if err == nil {
		//directory already exists, remove it
		err := os.RemoveAll(path)
		if err != nil {
			return err
		}
	}

	//create the directory
	err = os.MkdirAll(path, 0777)
	if err != nil {
		return err
	}

	return nil
}

func createZip(w *zip.Writer, dir, pathInZip string) error {
	//read through each file in build dir, adding it to zip
	//handle directories by recursive func calls.
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		filePath := filepath.Join(dir, f.Name())

		//recurse into directories
		//Build correct path to new directory for source file and path
		//for directory in zip file.
		if f.IsDir() {
			dirNew := filepath.Join(filePath, "/")
			pathInZip := filepath.Join(pathInZip, f.Name(), "/")
			createZip(w, dirNew, pathInZip)
		} else {
			fileToZip, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer fileToZip.Close()

			info, err := fileToZip.Stat()
			if err != nil {
				return err
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			//adds compression
			header.Method = zip.Deflate

			//set path to file in case file is located within subdirectory
			//This must be a relative path, with / representing the base of the zip
			header.Name = filepath.Join(pathInZip, f.Name())

			writer, err := w.CreateHeader(header)
			if err != nil {
				return err
			}

			_, err = io.Copy(writer, fileToZip)
			if err != nil {
				return err
			}
		} //end if: file or directory
	} //end for: loop through files to zip

	return nil
}

//
//
//

//copyDir recursively copies a directory tree, attempting to preserve permissions.
//Source directory must exist, destination directory must *not* exist.
//Symlinks are ignored and skipped.
//From https://gist.github.com/r0l1/92462b38df26839a3ca324697c8cba04
func copyDir(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		//remove dest if it already exists
		err := os.RemoveAll(dst)
		if err != nil {
			return fmt.Errorf("destination already exists, could not be automatically removed")
		}
	}

	err = os.MkdirAll(dst, 0777)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Skip symlinks.
			fileInfo, err := entry.Info()
			if err != nil {
				return err
			}

			if fileInfo.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//copyFile copies the contents of the file named src to the file named
//by dst. The file will be created if it does not already exist. If the
//destination file exists, all it's contents will be replaced by the contents
//of the source file. The file mode will be copied from the source and
//the copied data is synced/flushed to stable storage.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	err = out.Sync()
	if err != nil {
		return err
	}

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return err
	}

	return nil
}
