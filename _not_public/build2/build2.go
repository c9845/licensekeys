/*
Package build2 builds the distributable software. This builds the binary and gathers
other files into a directory, then zips the directory up for easy distributing.

This works off of GOPATH! This was necessary to have a consistent "base" path to use
to find the resourced noted in the config file to build or include in a distribution.

Note the 0777 permission on MkdirAll and Write. This is needed to get building to
work properly on MacOS. If 0755 permissions are used, "permission denied" errors
occur when reading created files. I am sure there is a fix for this, or setting
permissions to 0777 in only certain places, but I didn't want to figure this out.
*/
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v2"
)

// config holds configuration for this building. This is used to allow reuse of this
// build script for multiple apps without modifying the below code much, if at all.
type config struct {
	//Name is the name the binary will be built as as passed to `go build -o`.
	//.exe will be added for windows.
	Name string `yaml:"Name"`

	//Namespace is the subdirectory of GOAPTH where the file holding main() is found.
	//
	//It is necessary to locate this off of GOPATH because when we commit this file
	//to source control, we don't want to commit an absolute path that would (1) leak
	//user info on where this file is being run from, and (2) require changing this
	//field for each user running this file to include their system username. Basing
	//this off GOPATH just works for this case although it does require this repo to
	//be located in the GOPATH.
	//
	//Another option would be placing main.go in a subdirectory of the namespace, such
	//as cmd/ and placing the build script (this script) in the root of the namespace.
	//This would require fixing a lot of stuff (embedded paths, etc.) but could
	//alleviate the need to depend on GOPATH since we would use the build script's
	//directory as the base path to locate things.
	Namespace string `yaml:"Namespace"`

	//EntrypointFile is the name of the file that holds main() and is used to start
	//the app via `go run`. This is needed for `go run` to get the version from the
	//binary via the `-version` flag. Typically main.go.
	EntrypointFile string `yaml:"EntrypointFile"`

	//ChangelogPath is the path to the changelog file based off the namespace. This
	//will be used to compare the latest version noted in the changelog against the
	//version retreived from git and the binary. If blank, a version is not retrieved
	//from this file.
	ChangelogPath string `yaml:"ChangelogPath"`

	//GoBuildTags is anything passed to `go build -tags`.
	//"modernc" so we can build with SQLite statically.
	GoBuidTags string `yaml:"GoBuidTags"`

	//GoBuildTags is anything passed to `go build -ldflags`. Typically "-s -w".
	GoBuildLdFlags string `yaml:"GoBuildLdFlags"`

	//GoBuildTrimpath determines if the -trimpath flag should be passed to `go build`.
	GoBuildTrimpath bool `yaml:"GoBuildTrimpath"`

	//UseCGO determines if CGO should be enabled. Typically disabled to allow for
	//static builds.
	UseCGO bool `yaml:"UseCGO"`

	//OutputDir is the directory off of Namespace where builds will be saved to.
	OutputDir string `yaml:"OutputDir"`

	//IncludeDirs is a list of directories, based off Namespace, to include recursively
	//in zipped distributions.
	IncludeDirs []string `yaml:"IncludeDirs"`

	//IgnoreExtensions prevents copying of files in IncludeDirs that end with one of
	//these extensions. Used for skipping source code (ex.: .ts files but we want to
	//copy .js files).
	IgnoreExtensions []string `yaml:"IgnoreExtensions"`

	//IgnoreFiles prevents copying of specific files in IncludeDirs. See filepath.Match
	//for the use of wildcard and matching logic.
	IgnoreFiles []string `yaml:"IgnoreFiles"`

	//IncludeRootFiles is a list of files stored at the root of Namespace that should
	//be included with distributed builds. These files will be copied to the root of
	//the distributions. Typically copyright, license, readme.
	IncludeRootFiles []string `yaml:"IncludeRootFiles"`

	//PathToPandoc is the path to the Pandoc binary used to convert .md files to .txt
	//files. If set to -1, conversion will be skipped.
	PathToPandoc string `yaml:"PathToPandoc"`

	//PandocOutputFormat is the format of files to create from markdown inputs.
	//Ex.: pdf, txt/plain
	PandocOutputFormat string `yaml:"PandocOutputFormat"`

	//ZipDistributions determines whether or not to zip the distribution files into
	//a single zip file.
	ZipDistributions bool `yaml:"ZipDistributions"`
}

// osArch is an OS and CPU architecture thata golang binary can be built for.
type osArch struct {
	OS   string //windows, linux, darwin
	Arch string //usually "amd64"
}

// definedOSArchs is the list of OS and CPU architectures we support building binaries
// for. These will be use in GOOS and GOARCH.
var definedOSArchs = []osArch{
	{
		OS:   "windows",
		Arch: "amd64",
	},
	{
		OS:   "linux",
		Arch: "amd64",
	},
	{
		OS:   "darwin", //mac
		Arch: "amd64",
	},
	{
		OS:   "darwin", //mac
		Arch: "arm64",
	},
}

// Define command line flags.
var skipVersionCheck bool
var verbose bool
var configFilePath string

// Config file handling.
const defaultConfigFileName = "build2.yaml" //not using .conf so tab completion of "go build" works right... .conf is "less" than .go so it always gets completed first.

var cfg config

func init() {
	//Handle flags.
	flag.StringVar(&configFilePath, "config", "./"+defaultConfigFileName, "Full path to the configuration file.")
	flag.BoolVar(&skipVersionCheck, "skip-version-check", false, "Skip check to make sure git, binary, and changelog versions match.")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging.")
	flag.Parse()

	//Read the config file.
	configFilePath = filepath.Clean(configFilePath)
	if strings.TrimSpace(configFilePath) == "" {
		log.Fatalln("No path to config file provided.")
		return
	}

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		y, err := yaml.Marshal(defaultConfig)
		if err != nil {
			log.Fatalln("Could not marshal default config.", err)
			return
		}

		f, err := os.Create(configFilePath)
		if err != nil {
			log.Fatalln("Could not create file for default config.", err)
			return
		}
		defer f.Close()

		_, err = f.Write(y)
		if err != nil {
			log.Fatalln("Could not write default config to file.", err)
			return
		}

		log.Fatalln("Default config did not exist, it was created. Please update file before rerunning build2.")
		return
	}

	f, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatalln("Could not read config file from path.", err)
		return
	}

	err = yaml.Unmarshal(f, &cfg)
	if err != nil {
		log.Println(string(f))
		log.Fatalln("Could not parse config file as yaml.", err)
		return
	}

	cfg.validate()

	//Handle Pandoc path based on OS this script is running on.
	if cfg.PathToPandoc == "-1" {
		log.Println("WARNING! Pandoc conversion will be skipped since PathToPandoc is '-1'.")
	}

	//Diagnostics.
	if cfg.PathToPandoc != "-1" && cfg.PandocOutputFormat == "pdf" {
		log.Println("WARNING! Converting Markdown to PDF with Pandoc is slow. It also required wkhtmltopdf to be installed and accessible via command line by including the binary in your PATH.")
	}
}

func main() {
	//Get and commpare versions from git, binary, and changelog.
	version, err := getAndCompareVersions()
	if err != nil {
		log.Fatalln("Error getting version.", err)
		return
	}

	//Create directory where builds should be stored.
	buildsOuputDir := cfg.OutputDirAbs()
	err = os.MkdirAll(buildsOuputDir, 0777)
	if err != nil {
		log.Fatalln("Could not create build output directory.", err)
		return
	}

	//Create directory where common files distributed in builds will be stored. These
	//files are sent out with the distributed binaries and include static files,
	//install and help docs, and other files. The files are copied from source to this
	//common directory, then this common directory is copied to each distribution
	//being build. Doing this method of copying to each distribution relieves us from
	//having to copy files from the source each time for each distribution versus just
	//copying the common directory.
	commonDir := cfg.CommonDirAbs(version)
	err = createCommonDir(commonDir)
	if err != nil {
		log.Fatalln("Could not create temp common directory.", err)
		return
	}

	//Populate the common directory. This copies source directories listed in
	//IncludeDirs to the common directory that was just created. This also copies
	//files in IncludeRootFiles.
	err = populateCommonDir(commonDir)
	if err != nil {
		log.Fatalln("Could not copy files to common directory.", err)
		return
	}

	//Convert files in common directory from md to txt, if neeed.
	err = convertMarkDown(commonDir)
	if err != nil {
		log.Fatalln("Could not convert markdown files in common directory.", err)
		return
	}

	//Build the binaries and create zip distributions as needed.
	err = buildDistributions(definedOSArchs, version)
	if err != nil {
		log.Fatalln("Could not build distributions.", err)
		return
	}

	//Delete the common directory.
	err = os.RemoveAll(commonDir)
	if err != nil {
		log.Fatalln("Could not remove common directory after building.", err)
		return
	}

}

// NamespaceAbs returns the absolute path to Namespace.
func (c *config) NamespaceAbs() string {
	gopath := os.Getenv("GOPATH")
	return filepath.Join(gopath, "src", cfg.Namespace)
}

// OutputDirAbs returns the absolue path to OutputDir.
func (c *config) OutputDirAbs() string {
	return filepath.Join(c.NamespaceAbs(), c.OutputDir)
}

// CommonDirAbs returns the absolute path to the common directory for a version.
func (c *config) CommonDirAbs(version string) string {
	return filepath.Join(c.OutputDirAbs(), "common-"+version)
}

// getVersionFromGit returns the tagged version from git.
// See https://git-scm.com/docs/git-describe#:~:text=The%20%22g%22%20prefix%20stands%20for.
//
// Returns: <tag> which is <x.y.z>.
func getVersionFromGit() (version string, err error) {
	//Execute `git describe`.
	const command = "git"
	args := []string{"describe"}
	cmd := exec.Command(command, args...)

	out, err := cmd.Output()
	if err != nil {
		return
	}

	//Version from git may have some commit data appended to it, strip it.
	version = strings.TrimSpace(string(out))
	version = strings.TrimPrefix(version, "v")
	version, _, _ = strings.Cut(version, "-")

	return
}

// getVersionFromBinary gets the version from the -version flag passed to the binary.
// This simply runs "go run main.go" (or whatever the EntrypointFile is set to) in the
// Namespace directory.
//
// Returns: <x.y.z>.
func getVersionFromBinary() (version string, err error) {
	//Build full path to entrypoint file to run in `go run`.
	p := filepath.Join(cfg.NamespaceAbs(), cfg.EntrypointFile)

	//Execute `go run`.
	const command = "go"
	const versionFlag = "-version"
	args := []string{"run", p, versionFlag}
	cmd := exec.Command(command, args...)

	out, err := cmd.Output()
	if err != nil {
		log.Println(cmd.String())
		return
	}

	version = strings.TrimSpace(string(out))
	version = strings.TrimPrefix(version, "v")
	return
}

// getVersionFromChangelog gets the latest version from the changelog file.
//
// Returns: <x.y.z>.
func getVersionFromChangelog() (version string, err error) {
	//See if getting version from changelog should be skipped.
	if cfg.ChangelogPath == "" {
		return "", nil
	}

	//Build path to changelog file.
	p := filepath.Join(cfg.NamespaceAbs(), cfg.ChangelogPath)

	//Read the first/top line from the file.
	f, err := os.Open(p)
	if err != nil {
		return
	}
	defer f.Close()

	b := bufio.NewReader(f)
	version, err = b.ReadString('\n')
	if err != nil {
		return
	}

	//First line may include release date. Strip it.
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	version, _, _ = strings.Cut(version, " ")
	return
}

// getAndCompareVersions calls getVersionFromGit, getVersionFromBinary, and
// getVersionFromChangelog, compares the values to make sure the match, and returns
// the version if they do match.
func getAndCompareVersions() (version string, err error) {
	gitVersion, err := getVersionFromGit()
	if err != nil {
		return
	}

	binaryVersion, err := getVersionFromBinary()
	if err != nil {
		return
	}

	changelogVersion, err := getVersionFromChangelog()
	if err != nil {
		return
	}

	if verbose {
		log.Printf("Versions: %s (git), %s (binary), %s (changelog)", gitVersion, binaryVersion, changelogVersion)
	}

	if skipVersionCheck {
		log.Println("WARNING!", "Skipping version match check due to flag, using version from binary.")
	} else if gitVersion != binaryVersion {
		err = fmt.Errorf("version mismatch git: %s, binary: %s", gitVersion, binaryVersion)
		return
	} else if cfg.ChangelogPath != "" && changelogVersion != binaryVersion {
		err = fmt.Errorf("version mismatch changelog: %s, binary: %s", changelogVersion, binaryVersion)
		return
	} else if cfg.ChangelogPath != "" && gitVersion != changelogVersion {
		err = fmt.Errorf("version mismatch git: %s, changelog: %s", gitVersion, changelogVersion)
		return
	}

	return binaryVersion, nil
}

// createCommonDir creates the common directory where static files are stored before
// being incorporated into distributions.
func createCommonDir(commonDir string) (err error) {
	//Make sure the directory doesn't already exist. It may contain old source files.
	err = os.RemoveAll(commonDir)
	if err != nil {
		return
	}

	//Create the directory.
	err = os.MkdirAll(commonDir, 0777)
	return
}

// populateCommonDir copies files from source to the common directory.
func populateCommonDir(commonDir string) (err error) {
	//Copy each of the directories listed in IncludeDirs, recursively.
	//IgnoreExtensions and IgnoreFiles are ignored, obviously.
	for _, dir := range cfg.IncludeDirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}

		src := filepath.Join(cfg.NamespaceAbs(), dir)
		dst := filepath.Join(commonDir, dir)
		innerErr := copyDir(src, dst)
		if innerErr != nil {
			return innerErr
		}
	}

	//Copy files at root of namespace.
	for _, f := range cfg.IncludeRootFiles {
		src := filepath.Join(cfg.NamespaceAbs(), f)
		dst := filepath.Join(commonDir, f)

		//Create the destination file.
		dstFile, innerErr := os.Create(dst)
		if innerErr != nil {
			return innerErr
		}
		defer dstFile.Close()

		//Copy file to destination.
		srcFile, innerErr := os.Open(src)
		if innerErr != nil {
			return innerErr
		}
		defer srcFile.Close()

		_, innerErr = io.Copy(dstFile, srcFile)
		if innerErr != nil {
			return innerErr
		}
	}

	return
}

// copyDir recursively copies a directory tree.
func copyDir(sourceDir, commonDir string) (err error) {
	sourceDir = filepath.Clean(sourceDir)
	commonDir = filepath.Clean(commonDir)

	err = filepath.WalkDir(sourceDir, func(itemInSourcePath string, d fs.DirEntry, err error) error {
		//Error with path.
		if err != nil {
			return err
		}

		//Skip certain file extensions, i.e.: source code stuff.
		filename := filepath.Base(itemInSourcePath)
		extension := filepath.Ext(itemInSourcePath)
		if slices.Contains(cfg.IgnoreExtensions, extension) {
			return nil
		}

		//Skip certain files.
		for _, filenamePattern := range cfg.IgnoreFiles {
			match, err := filepath.Match(filenamePattern, filename)
			if err != nil {
				return err
			} else if match {
				return nil
			}
		}

		//Get path of currently being visited file/dir inside src.
		itemPartialPath := strings.Replace(itemInSourcePath, sourceDir, "", 1)

		//Get path to item as it should be in common directory.
		itemInCommonDirPath := filepath.Join(commonDir, itemPartialPath)

		//If we found a directory, create it in the common directory.
		if d.IsDir() {
			err := os.MkdirAll(itemInCommonDirPath, 0777)
			if err != nil {
				return err
			}
		} else {
			//Create the destination file.
			dstFile, err := os.Create(itemInCommonDirPath)
			if err != nil {
				return err
			}
			defer dstFile.Close()

			//Copy file to destination.
			srcFile, err := os.Open(itemInSourcePath)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			_, err = io.Copy(dstFile, srcFile)
			if err != nil {
				return err
			}
		}

		return nil
	})
	return
}

// convertMarkDown converts markdown files in the common directory to another format.
// This is done since reading markdown by end users is messy.
func convertMarkDown(commonDir string) (err error) {
	//Skip conversion if pandoc isn't provided/installed.
	if cfg.PathToPandoc == "-1" {
		return
	}

	//Walk the path, converting files found.
	//
	//This isn't done as part of copyDir, even though we use WalkDir there, since we
	//may want to skip Pandoc conversion.
	err = filepath.WalkDir(commonDir, func(inputPath string, d fs.DirEntry, err error) error {
		//Error with path.
		if err != nil {
			return err
		}

		//Skip directories.
		if d.IsDir() {
			return nil
		}

		//Get input info.
		inputExtension := filepath.Ext(inputPath)
		inputFilename := filepath.Base(inputPath)

		//Create output info.
		outputExtension := ""
		outputFilename := ""
		switch cfg.PandocOutputFormat {
		case "plain", "txt":
			outputExtension = ".txt"
		case "pdf":
			outputExtension = ".pdf"

		default:
			outputExtension = ""
			outputFilename = inputFilename
		}

		outputFilename = strings.Replace(inputFilename, ".md", outputExtension, 1)
		outputPath := filepath.Join(filepath.Dir(inputPath), outputFilename)

		//Skip non-markdown files. We don't convert these!
		if inputExtension != ".md" {
			return nil
		}

		//Convert via pandoc.
		command := cfg.PathToPandoc
		args := []string{
			"-f", "markdown",
			"-t", "plain",
			inputPath,
			"-o", outputPath,
		}

		if cfg.PandocOutputFormat == "pdf" {
			args = append(args, "-t", "html")
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd := exec.Command(command, args...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			log.Println("MD to TXT error (stdout): ", stdout.String())
			log.Println("MD to TXT error (stderr): ", stderr.String())
			log.Println("MD to TXT error (PathTo): ", cfg.PathToPandoc)
			log.Println("MD to TXT error (command):", command)
			log.Println("MD to TXT error (args):   ", args)
			log.Fatalln("MD to TXT error (err):    ", err)
			return err
		}

		//Remove the old/input markdown file. Only ship the converted file.
		err = os.Remove(inputPath)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return
	}

	return nil
}

// buildDistributions creates a directory for each osArch, builds the binary for the
// osArch, copies the common directory into this new directory, creates a checksum of
// the binary, and optionally zips the directory and checksums the zip as well. These
// are the distributable files you would send to clients.
//
// This runs `go build`.
func buildDistributions(osArches []osArch, version string) (err error) {
	for _, oa := range osArches {
		//Determine name for the distribution based on OS and CPU architecture.
		distributionDirName := cfg.Name + "-" + version + "-" + oa.OS + "_" + oa.Arch

		//Create the directory for this distribution.
		distributionDirAbs := filepath.Join(cfg.OutputDirAbs(), distributionDirName)
		innerErr := os.MkdirAll(distributionDirAbs, 0777)
		if innerErr != nil {
			return fmt.Errorf("error mkdir distribution %w", innerErr)
		}

		//Create name of the binary to build. Mostly to handle windows.
		binaryName := cfg.Name
		if oa.OS == "windows" {
			binaryName += ".exe"
		}

		//Name and place to output the built binary as.
		binaryPathAbs := filepath.Join(distributionDirAbs, binaryName)

		//Construct the arguments to the command.
		args := []string{
			"build",
			"-o", binaryPathAbs,
		}

		if len(cfg.GoBuidTags) > 0 {
			args = append(args, "-tags", cfg.GoBuidTags)
		}

		if len(cfg.GoBuildLdFlags) > 0 {
			args = append(args, "-ldflags", cfg.GoBuildLdFlags)
		}

		if cfg.GoBuildTrimpath {
			args = append(args, "-trimpath")
		}

		//Build command to run.
		const command = "go"
		cmd := exec.Command(command, args...)

		//Set the directory to run the command in. This is the root of the Namespace
		//where the file holding main() should be located.
		//
		//We could also specify the path the the file to build by appending it to the
		//args.
		cmd.Dir = cfg.NamespaceAbs()

		//Add modifier environmental variables to this command. These won't be set
		//permenently.
		cmd.Env = append(cmd.Environ(), "GOOS="+oa.OS, "GOARCH="+oa.Arch)
		if cfg.UseCGO {
			cmd.Env = append(cmd.Environ(), "CGO_ENABLED=1")
		} else {
			cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0")
		}

		//Run the build command. Same as `go build`.
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if verbose {
			log.Printf("Building...(%s, %s) %s %s", oa.OS, oa.Arch, command, strings.Join(args, " "))
		} else {
			log.Printf("Building...(%s, %s)", oa.OS, oa.Arch)
		}
		innerErr = cmd.Run()
		if innerErr != nil {
			log.Println("Build error (stdout):", stdout.String())
			log.Println("Build error (stderr):", stderr.String())
			log.Fatalln("Build error (err):   ", innerErr)
			return innerErr
		}

		//Create checksum of binary and save to file. Have to make sure file doesn't
		//already exist because hash could be wrong and old!
		binaryChecksumPathAbs := binaryPathAbs + ".sha256"
		innerErr = os.RemoveAll(binaryChecksumPathAbs)
		if innerErr != nil {
			return fmt.Errorf("error removing old binary checksum %w", innerErr)
		}

		binary, innerErr := os.Open(binaryPathAbs)
		if innerErr != nil {
			return fmt.Errorf("error opening binary for hashing %w", innerErr)
		}
		defer binary.Close()

		h := sha256.New()
		_, innerErr = io.Copy(h, binary)
		if innerErr != nil {
			return fmt.Errorf("error copying binary checksum %w", innerErr)
		}

		//Format hash in specific format to work with "sha256sum --check" command on
		//linux systems. Note the two spaces to separate hash and filename!
		hash := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
		hashFormatted := hash + "  " + filepath.Base(binaryName)

		innerErr = os.WriteFile(binaryChecksumPathAbs, []byte(hashFormatted), 0777)
		if innerErr != nil {
			return fmt.Errorf("error writing binary hash %w", innerErr)
		}

		//Close acces to the binary since we are done using it. Need to do this to
		//allow deleting of the binary in the source un-zipped files, if needed.
		binary.Close()

		//Copy common files from common directory to the build directory.
		innerErr = copyDir(cfg.CommonDirAbs(version), distributionDirAbs)
		if innerErr != nil {
			return fmt.Errorf("error copying common dir %w", innerErr)
		}

		//Check if we should zip the file.
		if !cfg.ZipDistributions {
			continue
		}

		//Make sure zip file doesn't already exist.
		zipFileAbs := distributionDirAbs + ".zip"
		innerErr = os.RemoveAll(zipFileAbs)
		if innerErr != nil {
			return fmt.Errorf("error removing old zip %w", innerErr)
		}

		//Zip the distribution up.
		zipFile, innerErr := os.Create(zipFileAbs)
		if innerErr != nil {
			return fmt.Errorf("error creating zip file %w", innerErr)
		}

		zw := zip.NewWriter(zipFile)
		defer zw.Close()

		sourceDir := os.DirFS(distributionDirAbs)
		innerErr = zw.AddFS(sourceDir)
		if innerErr != nil {
			return fmt.Errorf("error creating zip %w", innerErr)
		}

		//Close the zipper since it was created successfully. Need to do this before
		//reading zip to create checksum.
		zw.Close()

		//Delete the source un-zipped directory of files.
		innerErr = os.RemoveAll(distributionDirAbs)
		if innerErr != nil {
			return fmt.Errorf("error deleting unzipped dir %w", innerErr)
		}

		//Create checksum of the zip file and save to file.
		zipChecksumPath := zipFileAbs + ".sha256"
		innerErr = os.RemoveAll(zipChecksumPath)
		if innerErr != nil {
			return fmt.Errorf("error removing zip checksum %w", innerErr)
		}

		zf, innerErr := os.Open(zipFileAbs)
		if innerErr != nil {
			return fmt.Errorf("error opening zip for hashing %w", innerErr)
		}
		defer zf.Close()

		h = sha256.New()
		_, innerErr = io.Copy(h, zf)
		if innerErr != nil {
			return fmt.Errorf("error copying zip checksum %w", innerErr)
		}

		//Format hash in specific format to work with "sha256sum --check" command on
		//linux systems. Note the two spaces to separate hash and filename!
		hash = strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
		hashFormatted = hash + "  " + filepath.Base(zipFileAbs)

		innerErr = os.WriteFile(zipChecksumPath, []byte(hashFormatted), 0777)
		if innerErr != nil {
			return fmt.Errorf("error saving zip checksum %w", innerErr)
		}

		//This distribution is done, move to next os/arch.

	} //end for: loop through osArches

	return
}

// validate does some sanitizing and validation of the parsed config file.
func (c *config) validate() {
	c.Name = strings.TrimSpace(c.Name)
	c.Namespace = strings.TrimSpace(c.Namespace)
	c.EntrypointFile = strings.TrimSpace(c.EntrypointFile)
	c.ChangelogPath = strings.TrimSpace(c.ChangelogPath)
	c.GoBuidTags = strings.TrimSpace(c.GoBuidTags)
	c.GoBuildLdFlags = strings.TrimSpace(c.GoBuildLdFlags)
	c.OutputDir = strings.TrimSpace(c.OutputDir)
	c.PathToPandoc = strings.TrimSpace(c.PathToPandoc)
	c.PandocOutputFormat = strings.TrimSpace(c.PandocOutputFormat)

	c.Namespace = filepath.Clean(filepath.ToSlash(c.Namespace))
	c.ChangelogPath = filepath.Clean(filepath.ToSlash(c.ChangelogPath))
}

// defaultConfig is used to write a default config to a file when this script is called
// with a non-existent config file.
var defaultConfig = config{
	Name:             "your-app",
	Namespace:        "path/to/app/repo",
	EntrypointFile:   "main.go",
	ChangelogPath:    "_documentation/changelog.txt",
	GoBuidTags:       "modernc",
	GoBuildLdFlags:   "-s -w",
	GoBuildTrimpath:  true,
	UseCGO:           false,
	OutputDir:        "_builds",
	IncludeDirs:      []string{"_documentation"},
	IgnoreExtensions: []string{".ts"},
	IgnoreFiles: []string{
		"script.js",
		"*.script.min.js",
		"styles.css",
		"*.styles.min.css",
	},
	IncludeRootFiles: []string{
		"COPYRIGHT.md",
		"README.md",
	},
	PathToPandoc:       "-1",
	PandocOutputFormat: "pdf",
	ZipDistributions:   true,
}
