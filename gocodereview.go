package main

//Script used to scan the specified input YAML file for
//signature and run the scan
import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

// DefLinesBefore - Lines to display before the match
const DefLinesBefore = "2"

// DefLinesAfter - Lines to display after the match
const DefLinesAfter = "2"

// DefaultGrepBinPath path - Assume that it is hosted on Grep Bin
const DefaultGrepBinPath = "grep"

// Format for an Example YAML signature files
type signFileStruct struct {
	ID       string     `yaml:"id"`
	Name     string     `yaml:"name"`
	Author   string     `yaml:"author"`
	Severity string     `yaml:"severity"`
	Checks   []sigCheck `yaml:"checks"`
}

// Define a separate struct for checks
type sigCheck struct {
	Outfile string   `yaml:"outfile"`
	Regex   []string `yaml:"regex"`
	Notes   string   `yaml:"notes"`
}

// SIGFILEEXT - Extensions for YAML files
var SIGFILEEXT []string = []string{".yml", ".yaml"}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

// Find files that have the relevant extensions. By default, YAML is used.
func findSigFiles(filesToParse []string) []string {

	var sigFiles []string

	for _, fileToCheck := range filesToParse {
		for _, ext := range SIGFILEEXT {
			isSigFile := strings.Index(fileToCheck, ext)
			if isSigFile != -1 {
				sigFiles = append(sigFiles, fileToCheck)
				break
			}
		}
	}
	return sigFiles
}

// Parse the signature file given the struct and return the contents of YAML
// Signature file
func parseSigFile(sigFile string) signFileStruct {
	var sigFileContent signFileStruct
	yamlFile, err := ioutil.ReadFile(sigFile)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &sigFileContent)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return sigFileContent
}

// Execute a command to get the output, error. Command is executed when in the
// optionally specified 'cmdDir' OR it is executed with the current working dir
func execCmd(cmdToExec string) string {

	log.Printf("[v] Executing cmd: %s\n", cmdToExec)

	cmd := exec.Command("/bin/bash", "-c", cmdToExec)
	out, err := cmd.CombinedOutput()
	var outStr, errStr string
	if out == nil {
		outStr = ""
	} else {
		outStr = string(out)
	}

	if err == nil {
		errStr = ""
	} else {
		errStr = string(err.Error())
	}

	totalOut := (outStr + "\n" + errStr)

	return totalOut
}

// Execute a Grep Search for a particular regex on a folder
func execGrepSearch(grepBin string, folderToScan string, regex string,
	excludePatterns []string) string {
	cmd := "{grepBin} --color=always -E -i -r -n -A {DefLinesAfter} -B {DefLinesBefore} \"{regex}\" \"{folderToScan}\""
	cmd = strings.ReplaceAll(cmd, "{grepBin}", grepBin)
	cmd = strings.ReplaceAll(cmd, "{regex}", regex)
	cmd = strings.ReplaceAll(cmd, "{folderToScan}", folderToScan)
	cmd = strings.ReplaceAll(cmd, "{DefLinesAfter}", DefLinesAfter)
	cmd = strings.ReplaceAll(cmd, "{DefLinesBefore}", DefLinesBefore)

	if excludePatterns != nil {
		for _, excludePattern := range excludePatterns {
			if excludePattern != "" {
				cmd += " --exclude " + excludePattern
			}
		}
	}

	return execCmd(cmd)
}

// Worker function parses each YAML signature file, runs a search for the regexes
// on the folder to scan via the specified grep binary. The specified
// `excludePatterns` defines the paths to ignore when running the grep search
func worker(sigFileContents map[string]signFileStruct, sigFiles chan string,
	grepBin string, folderToScan string, excludePatterns []string,
	wg *sync.WaitGroup) {

	// Need to let the waitgroup know that the function is done at the end...
	defer wg.Done()

	// Check each signature on the folder to scan
	for sigFile := range sigFiles {

		log.Printf("Testing sigfile: %s on folder: %s\n", sigFile, folderToScan)

		// Get the signature file content previously opened and read
		sigFileContent := sigFileContents[sigFile]

		// First get the list of all checks to perform from file
		myChecks := sigFileContent.Checks

		for _, myCheck := range myChecks {

			cmdsOutput := ""

			// Get all the defined regexes for this check
			regexes := myCheck.Regex

			for _, regex := range regexes {
				cmdsOutput += execGrepSearch(grepBin, folderToScan, regex,
					excludePatterns) + "\n"
			}

			// Are there any special notes? Write them to the output
			notes := myCheck.Notes
			if notes != "" {
				cmdsOutput += "\n[!] " + notes
			}

			// Check if we need to store the output in an  output file
			outfile := myCheck.Outfile
			if outfile != "" {

				// Get the command and web request output together to write to file
				contentToWrite := cmdsOutput + "\n"

				// Write output to file
				ioutil.WriteFile(outfile, []byte(contentToWrite), 0644)

				// Let user know that we wrote results to an output file
				log.Printf("[*] Wrote results to outfile: %s\n", outfile)

			}
		}
	}
	//log.Printf("Completed check on path: %s\n", target["basepath"])
}

func main() {
	pathsWithSigFiles := flag.String("s", "",
		"Files/folders/file-glob patterns, containing YAML signature files")
	verbosePtr := flag.Bool("v", false, "Show commands as executed+output")
	maxThreadsPtr := flag.Int("mt", 20, "Max number of goroutines to launch")
	grepBinPtr := flag.String("gb", DefaultGrepBinPath,
		"Default 'grep' binary path")
	excludePtr := flag.String("e", "", "Exclude file e.g. *.js")
	folderToScanPtr := flag.String("f", "", "File or Folder to scan")

	flag.Parse()

	maxThreads := *maxThreadsPtr
	grepBin := *grepBinPtr
	exclude := *excludePtr
	folderToScan := *folderToScanPtr

	// Check if logging should be enabled
	verbose := *verbosePtr
	if !verbose {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

	// Check if folder to scan provided
	if folderToScan == "" {
		fmt.Printf("[-] Folder to scan must be provided\n")
		log.Fatalf("[-] Folder to scan must be provided")
	}

	// Check if folder to scan exists
	_, err := os.Stat(folderToScan)
	if err != nil {
		fmt.Printf("[-] Cannot read file/folder: %s. Does not exist\n",
			folderToScan)
		log.Fatalf("[-] Cannot read file/folder: %s. Does not exist",
			folderToScan)
	}

	if *pathsWithSigFiles == "" {
		fmt.Println("[-] Signature files must be provided.")
		log.Fatalf("[-] Signature files must be provided.")
	}

	log.Println("Convert the comma-sep list of files, folders to loop through")
	pathsToCheck := strings.Split(*pathsWithSigFiles, ",")

	// List of all files in the folders/files above
	var filesToParse []string

	log.Println("Loop through each path to to discover all files")
	for _, pathToCheck := range pathsToCheck {
		// Check if glob file-pattern provided
		log.Printf("Reviewing path: %s\n", pathToCheck)
		if strings.Index(pathToCheck, "*") >= 0 {
			matchingPaths, _ := filepath.Glob(pathToCheck)
			for _, matchingPath := range matchingPaths {
				filesToParse = append(filesToParse, matchingPath)
			}

		} else {

			//Check if file path exists
			fi, err := os.Stat(pathToCheck)
			if err != nil {
				log.Fatalf("[-] Path: %s not found\n", pathToCheck)
			} else {
				switch mode := fi.Mode(); {

				// Add all files from the directory
				case mode.IsDir():
					filepath.Walk(pathToCheck,
						func(path string, f os.FileInfo, err error) error {
							// Determine if the path is actually a file
							fi, err := os.Stat(path)
							if fi.Mode().IsRegular() == true {

								// Add the path if it doesn't already exist to list
								// of all files
								_, found := Find(filesToParse, path)
								if !found {
									filesToParse = append(filesToParse, path)
								}
							}
							return nil
						})

				// Add a single file, if not already present
				case mode.IsRegular():

					// Add the path if it doesn't already exist to list
					// of all files
					_, found := Find(filesToParse, pathToCheck)
					if !found {
						filesToParse = append(filesToParse, pathToCheck)
					}
				}
			}
		}
	}

	log.Printf("Total number of files: %d\n", len(filesToParse))

	// Get all the Yaml files filtered based on the extension
	sigFiles := findSigFiles(filesToParse)

	log.Printf("Number of signature  files: %d\n", len(sigFiles))

	// parse information from each signature file and store it so it doesn't
	// have to be read again & again
	sigFileContents := make(map[string]signFileStruct, len(sigFiles))
	for _, sigFile := range sigFiles {
		log.Printf("Parsing signature file: %s\n", sigFile)
		sigFileContents[sigFile] = parseSigFile(sigFile)
	}

	// Parse files to exclude
	excludePatterns := strings.Split(exclude, ",")

	// Channel of Signature files to parse to each thread to process
	sigFilesChan := make(chan string)

	// Starting max number of concurrency threads
	var wg sync.WaitGroup
	for i := 1; i <= maxThreads; i++ {
		wg.Add(1)

		log.Printf("Launching goroutine: %d for assessing folder: %s\n", i,
			folderToScan)
		go worker(sigFileContents, sigFilesChan, grepBin, folderToScan,
			excludePatterns, &wg)
	}

	// Loop through each signature file and pass it to each thread to process
	for _, sigFile := range sigFiles {
		sigFilesChan <- sigFile
	}

	close(sigFilesChan)

	// Wait for all threads to finish processing the regex checks
	wg.Wait()
}
