package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
	}
	submodule := flag.Arg(0)

	version, err := gitVersion()
	if err != nil {
		fatal(err)
	}

	// Check that version >= 1.8.5
	if version[0] < 2 && ((version[1] == 8 && version[2] < 5) || version[1] < 8) {
		fatalf("git-rm-submodule needs git version >= 1.8.5\n")
	}

	gitOrFatal("submodule", "deinit", "-f", submodule)
	gitOrFatal("rm", "-rf", submodule)

	topLevel := revParse("--show-toplevel")
	subPath, err := submodulePath(topLevel, submodule)
	if err != nil {
		fatal(err)
	}
	modulePath := filepath.Join(revParse("--git-dir"), "modules", subPath)
	if err := os.RemoveAll(modulePath); err != nil {
		fatal(err)
	}
}

func usage() {
	fmt.Println(`usage: git rm-submodule <submodule>

This command will deinit a submodule and delete all associated files such that
the submodule is removed from git completely. It is equivalent to performing
the following steps:

1. git submodule deinit -f path/to/submodule
2. git rm -rf path/to/submodule
3. rm -rf .git/modules/path/to/submodule

git-rm-submodule uses git deinit (introduced in 1.8.3) and certain behavior
of git add changed in 1.8.5, so it fails unless git has version at least
1.8.5.

Note that you will probably want to commit these changes afterwards.
`)
	os.Exit(129) // 129 is used by git commands for -h or incorrect usage
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func gitVersion() ([3]int, error) {
	var v [3]int
	output, err := git("version")
	if err != nil {
		return v, err
	}
	vstring := strings.TrimSpace(strings.TrimPrefix(string(output), "git version"))
	parts := strings.Split(vstring, ".")
	if len(parts) != 3 {
		return v, fmt.Errorf("version did not have 3 parts")
	}
	for i := range v {
		v[i], err = strconv.Atoi(parts[i])
		if err != nil {
			return v, err
		}
	}
	return v, nil
}

func submodulePath(toplevel, sub string) (string, error) {
	abs, err := filepath.Abs(sub)
	if err != nil {
		return "", err
	}
	return filepath.Rel(toplevel, abs)
}

func git(args ...string) (output []byte, err error) {
	return exec.Command("git", args...).CombinedOutput()
}

func revParse(arg string) string {
	return strings.TrimSpace(string(gitOrFatal("rev-parse", arg)))
}

func gitOrFatal(args ...string) []byte {
	iargs := make([]interface{}, len(args))
	for i := range iargs {
		iargs[i] = args[i]
	}
	output, err := git(args...)
	if err != nil {
		fatalf("Error running git %s\n%s", fmt.Sprintln(iargs...), string(output))
	}
	return output
}
