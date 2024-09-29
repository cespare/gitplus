package main

import (
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
)

func main() {
	log.SetFlags(0)

	// If gitplus was invoked as git-foo where foo is one of the gitplus
	// commands, run it directly.
	if name, ok := strings.CutPrefix(filepath.Base(os.Args[0]), "git-"); ok {
		if c, ok := commands[name]; ok {
			c.run(os.Args[1:])
			return
		}
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(129)
	}
	if c, ok := commands[os.Args[1]]; ok {
		c.run(os.Args[2:])
		return
	}
	switch os.Args[1] {
	case "help", "-h", "--help":
		usage()
		return
	}
	usage()
	os.Exit(129)
}

type command struct {
	desc string
	run  func([]string)
}

var commands = map[string]command{
	"rename-branch": {
		desc: "rename a local branch along with its tracking branch",
		run:  cmdRenameBranch,
	},
}

func usage() {
	fmt.Fprint(os.Stderr, "usage: gitplus <command>\n\nThe commands are:\n\n")
	tw := tabwriter.NewWriter(os.Stderr, 0, 0, 4, ' ', 0)
	for _, cmd := range slices.Sorted(maps.Keys(commands)) {
		c := commands[cmd]
		fmt.Fprintf(tw, "  %s\t%s\t\n", cmd, c.desc)
	}
	tw.Flush()
	fmt.Fprint(os.Stderr, `
Run 'gitplus <command> -h' for more information about a command.

Another way to use gitplus is to install symlinks in your $PATH with names like
git-rename-branch that point at the gitplus binary. Then instead of

    gitplus rename-branch args...

you can use

    git rename-branch args...
`)
}
