package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/scanner"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/tools/imports"
)

var (
	write      = flag.Bool("w", false, "write result to (source) file instead of stdout")
	list       = flag.Bool("l", false, "list files whose formatting differs from goimport's")
	doDiff     = flag.Bool("d", false, "display diffs instead of rewriting files")
	allErrors  = flag.Bool("e", false, "report all errors (not just the first 10 on different lines)")
	localPkg   = flag.String("local", "", "put imports beginning with this string after 3rd-party packages; comma-separated list")
	formatOnly = flag.Bool("format-only", false, "if true, don't fix imports and only format")
	srcDir     = flag.String("srcdir", "", "choose imports as if source code is from `dir`")
	verbose    = flag.Bool("v", false, "verbose logging")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: goimports-all [flags] [path ...]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 {
		if err := processFile("<standard input>", os.Stdin, os.Stdout); err != nil {
			report(err)
		}
		os.Exit(exitCode)
	}

	for i := range flag.NArg() {
		path := flag.Arg(i)
		if err := processPath(path); err != nil {
			report(err)
		}
	}
	os.Exit(exitCode)
}

var exitCode = 0

func report(err error) {
	scanner.PrintError(os.Stderr, err)
	exitCode = 2
}

func processPath(path string) error {
	// Handle ./... pattern
	if strings.HasSuffix(path, "/...") || path == "..." {
		dir := strings.TrimSuffix(path, "/...")
		if dir == "" || dir == "." {
			dir = "."
		}
		return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				// Skip vendor and hidden directories
				if info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !isGoFile(info) {
				return nil
			}
			if err := processFile(path, nil, os.Stdout); err != nil {
				report(err)
			}
			return nil
		})
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !isGoFile(info) {
				return nil
			}
			if err := processFile(path, nil, os.Stdout); err != nil {
				report(err)
			}
			return nil
		})
	}

	return processFile(path, nil, os.Stdout)
}

func isGoFile(f os.FileInfo) bool {
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

func processFile(filename string, in io.Reader, out io.Writer) error {
	if *verbose {
		fmt.Fprintf(os.Stderr, "processing %s\n", filename)
	}

	var src []byte
	var err error

	if in != nil {
		src, err = io.ReadAll(in)
	} else {
		src, err = os.ReadFile(filename)
	}
	if err != nil {
		return err
	}

	opt := &imports.Options{
		TabWidth:   8,
		TabIndent:  true,
		Comments:   true,
		Fragment:   true,
		FormatOnly: *formatOnly,
		AllErrors:  *allErrors,
	}

	if *localPkg != "" {
		imports.LocalPrefix = *localPkg
	}

	target := filename
	if *srcDir != "" {
		target = filepath.Join(*srcDir, filepath.Base(filename))
	}

	res, err := imports.Process(target, src, opt)
	if err != nil {
		return err
	}

	if !bytes.Equal(src, res) {
		if *list {
			if _, err := fmt.Fprintln(out, filename); err != nil {
				return err
			}
		}
		if *write {
			if err := os.WriteFile(filename, res, 0644); err != nil {
				return err
			}
		}
		if *doDiff {
			data, err := diff(src, res, filename)
			if err != nil {
				return err
			}
			if _, err := out.Write(data); err != nil {
				return err
			}
		}
	}

	if !*list && !*write && !*doDiff {
		_, err = out.Write(res)
	}

	return err
}

func diff(b1, b2 []byte, filename string) ([]byte, error) {
	f1, err := os.CreateTemp("", "goimports")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(f1.Name()) }()
	defer func() { _ = f1.Close() }()

	f2, err := os.CreateTemp("", "goimports")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(f2.Name()) }()
	defer func() { _ = f2.Close() }()

	if _, err := f1.Write(b1); err != nil {
		return nil, err
	}
	if _, err := f2.Write(b2); err != nil {
		return nil, err
	}

	data, err := exec.Command("diff", "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		return replaceTempFilename(data, filename)
	}
	return nil, err
}

func replaceTempFilename(diff []byte, filename string) ([]byte, error) {
	bs := bytes.SplitN(diff, []byte{'\n'}, 3)
	if len(bs) < 3 {
		return nil, fmt.Errorf("unexpected diff output")
	}
	// Replace temp filenames with actual filename
	bs[0] = []byte("--- a/" + filename)
	bs[1] = []byte("+++ b/" + filename)
	return bytes.Join(bs, []byte{'\n'}), nil
}
