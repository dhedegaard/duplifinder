package main

import (
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// Defines the default number of bytes to hash when comparing files.
const HASH_MAX = 1024 * 1024

func log(msg ...interface{}) {
	fmt.Fprintln(os.Stderr, msg)
}

func parsePath(dirname string, filesizes map[int64][]string) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return
	}
	for _, file := range files {
		filename := path.Join(dirname, file.Name())
		// If file is a directory, recurse.
		if file.IsDir() {
			parsePath(filename, filesizes)
			continue
		}

		// If it's a file, add it to the map.
		filesize := file.Size()
		filesizes[filesize] = append(filesizes[filesize], filename)
	}
}

func stringInSlice(obj string, slice []string) bool {
	for _, e := range slice {
		if e == obj {
			return true
		}
	}
	return false
}

func hashFile(filename string) (result []byte, err error) {
	// Initiate an empty buffer.
	buffer := make([]byte, HASH_MAX)

	// Open the file.
	fd, err := os.Open(filename)
	if err != nil {
		return buffer, errors.New(fmt.Sprint("Skipping: \"", filename, "\", unable to open."))
	}
	defer fd.Close()

	// Fetch up to the first HASH_MAX bytes from the file descriptor.
	_, err = fd.Read(buffer)
	if err != nil {
		return buffer, errors.New(fmt.Sprintf("Skipping: \"", filename, "\", unable to read."))
	}

	// Hash the data, and return it.
	hash := md5.New()
	return hash.Sum(buffer), nil
}

func main() {
	// Validate arguments.
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	errors := false

	filesizes := make(map[int64][]string, 0)
	dirnames := make([]string, 0)

	// Iterate on the directories as arguments.
	for _, dirname := range flag.Args() {
		// skip all but the first iteration on the same directory.
		if stringInSlice(dirname, dirnames) {
			continue
		}
		dirnames = append(dirnames, dirname)

		// Open the directory
		file, err := os.Open(dirname)
		if err != nil {
			log("unable to open directory \"", dirname, "\", skipping...")
			errors = true
			continue
		}
		// Stat the file descriptor.
		fi, err := file.Stat()
		if err != nil {
			log("Unable to stat file \"", dirname, "\", skipping...")
			errors = true
			continue
		}
		// Check if the supposed directory really is a directory.
		if !fi.Mode().IsDir() {
			log("Dirname \"", dirname, "\" is not a directory, skipping...")
			errors = true
			continue
		}
		file.Close()

		// Recurse on the dirname.
		parsePath(dirname, filesizes)
	}

	// Hash all files with equal sizes, check for equal hashes means equal files.
	hashing := make(map[[HASH_MAX]byte][]string, 0)
	for _, filenames := range filesizes {
		if len(filenames) < 2 {
			continue
		}

		for _, filename := range filenames {
			hashedSlice, err := hashFile(filename)
			if err != nil {
				log(err)
				continue
			}

			// Convert slice to an array.
			var hashedArray [HASH_MAX]byte
			copy(hashedArray[:], hashedSlice[0:HASH_MAX])

			// Append it to the hashed result.
			hashing[hashedArray] = append(hashing[hashedArray], filename)
		}
	}

	// List all the double files found.
	fmt.Println("The following files are duplicates of eachother:")
	totalcount := 0
	for _, filenames := range hashing {
		filecount := len(filenames)

		// Skip hashes with only 1 filename.
		if filecount < 2 {
			continue
		}

		fmt.Printf("  [%d] \"%s\"\n", filecount, strings.Join(filenames, "\", \""))
		totalcount++
	}

	// Notify the user, if no duplicates were found.
	if totalcount == 0 {
		fmt.Println("No duplicates found!")
	}

	if errors {
		os.Exit(1)
	}
}
