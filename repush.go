package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

func cmdRepush(args []string) {
	fs := flag.NewFlagSet("repush", flag.ExitOnError)
	verbose := fs.Bool("v", false, "Verbose mode")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `usage: git repush [flags...] <base> [branch]

Flags:

`)
		fs.PrintDefaults()
		fmt.Fprint(os.Stderr, `
Repush performs the "rebase merge" worfklow with retries. That is, it:

- Pulls the base branch
- Rebases branch onto the base
- Pushes the branch
- Fast-forwards base to the branch
- Pushes the base
- If the push fails, retry all preceding steps
- If the branch tracks a remote branch of the same name, deletes it
- Deletes the branch locally

Typically the base branch is the main branch and the repushed branch is a
feature branch.

If a branch name is given, repush first performs a 'git switch <branch>' before
proceding. If a branch name is not given, the current branch is used.
`)
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		log.Println("Need branch name to repush")
		fs.Usage()
		os.Exit(129)
	}
	base := fs.Arg(0)
	var branch string
	switch fs.NArg() {
	case 1:
	case 2:
		branch = fs.Arg(1)
	default:
		log.Println("Too many arguments")
		fs.Usage()
		os.Exit(129)
	}

	var logger *log.Logger
	if *verbose {
		logger = log.New(os.Stderr, "", 0)
	} else {
		logger = log.New(io.Discard, "", 0)
	}

	if err := repush(logger, base, branch); err != nil {
		log.Fatalln("Error:", err)
	}
}

func repush(logger *log.Logger, base, branch string) error {
	if branch == "" {
		var err error
		branch, err = currentBranch(logger)
		if err != nil {
			return err
		}
	} else {
		if _, err := runGit(logger, "switch", branch); err != nil {
			return err
		}
	}

	if base == branch {
		return errors.New("cannot repush a branch into itself")
	}

	_, behind, err := aheadBehind(logger, branch, branch+"@{upstream}")
	if err != nil {
		return err
	}
	if behind > 0 {
		return fmt.Errorf(
			"branch %s is behind upstream by %d commits",
			branch, behind,
		)
	}
	ahead, _, err := aheadBehind(logger, base, base+"@{upstream}")
	if err != nil {
		return err
	}
	if ahead > 0 {
		return fmt.Errorf(
			"base branch %s is ahead of its upstream by %d commits",
			base, ahead,
		)
	}

	if _, err := runGit(logger, "switch", base); err != nil {
		return err
	}
	if _, err := runGit(logger, "pull", "--ff-only"); err != nil {
		return err
	}
	if _, err := runGit(logger, "switch", branch); err != nil {
		return err
	}
	if _, err := runGit(logger, "rebase", base); err != nil {
		return err
	}
	if _, err := runGit(logger, "push"); err != nil {
		return err
	}
	if _, err := runGit(logger, "switch", base); err != nil {
		return err
	}
	if _, err := runGit(logger, "merge", "--ff-only", branch); err != nil {
		return err
	}
	if _, err := runGit(logger, "push"); err != nil {
		// FIXME: retry
		return err
	}

	panic("TODO")
}
