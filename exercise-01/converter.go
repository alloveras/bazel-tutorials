// A simple JSON to YAML converter.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

func CanonicalizePath(cwd, p string) string {
	if path.IsAbs(p) {
		return path.Clean(p)
	}
	return path.Clean(path.Join(cwd, p))
}

func Convert(r io.Reader, w io.Writer) error {
	var data interface{}
	dec := json.NewDecoder(r)
	enc := yaml.NewEncoder(w)

	defer enc.Close() // Force flush.

	if err := dec.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode input JSON: %w", err)
	}

	return enc.Encode(data)
}

func main() {
	var inputFile string
	var outputFile string

	flag.StringVar(&inputFile, "i", "", "The input JSON file.")
	flag.StringVar(&outputFile, "o", "", "The output YAML file.")

	flag.Parse()

	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "[ERROR]: An input file must be specified")
		os.Exit(1)
	}

	if outputFile == "" {
		fmt.Fprintln(os.Stderr, "[ERROR]: An output file must be specified")
		os.Exit(1)
	}

	// See: https://bazel.build/docs/user-manual#running-executables
	cwd := os.Getenv("BUILD_WORKING_DIRECTORY")

	inFile, err := os.Open(CanonicalizePath(cwd, inputFile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR]: Failed to open input file: %s\n", err)
		os.Exit(1)
	}
	defer inFile.Close()

	outFile, err := os.Create(CanonicalizePath(cwd, outputFile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR]: Failed to open output file: %s\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	if err := Convert(inFile, outFile); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR]: Failed to perform JSON to YAML conversion: %s\n", err)
		os.Exit(1)
	}
}
