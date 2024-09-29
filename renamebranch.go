package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"
)

func cmdRenameBranch(args []string) {
	fs := flag.NewFlagSet("rename-branch", flag.ExitOnError)
	verbose := fs.Bool("v", false, "Verbose mode")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `usage: git rename-branch [flags...] [old-name] <new-name>

Flags:

`)
		fs.PrintDefaults()
		fmt.Fprint(os.Stderr, `
This command renames a local branch using 'git branch -m'. Then it renames a
remote tracking branch, if any.

If old-name is not given, rename-branch renames the current branch.
`)
	}
	fs.Parse(args)

	var oldName, newName string
	switch fs.NArg() {
	case 1:
		newName = fs.Arg(0)
	case 2:
		oldName = fs.Arg(0)
		newName = fs.Arg(1)
	default:
		log.Println("Need one or two branch names")
		fs.Usage()
		os.Exit(129)
	}

	var logger *log.Logger
	if *verbose {
		logger = log.New(os.Stderr, "", 0)
	} else {
		logger = log.New(io.Discard, "", 0)
	}

	if err := rename(logger, oldName, newName); err != nil {
		log.Fatalln("Error:", err)
	}
}

func rename(logger *log.Logger, oldName, newName string) error {
	if newName == "" {
		return errors.New("need a non-empty target branch name")
	}
	if oldName == "" {
		var err error
		oldName, err = currentBranch(logger)
		if err != nil {
			return err
		}
	}
	if oldName == newName {
		// git branch -m lets you do this, but it's probably a mistake
		// and it makes subsequent code simpler to not have to think
		// about it.
		return errors.New("old and new branch names are the same")
	}
	logger.Printf("Renaming %s to %s", oldName, newName)
	_, err := runGit(logger, "branch", "-m", oldName, newName)
	if err != nil {
		return err
	}
	upstreamRef := fmt.Sprintf("%s@{upstream}", newName)
	upstream, err := runGit(logger, "rev-parse", "--abbrev-ref", upstreamRef)
	if err != nil {
		// We know the branch exists, so just assume any error is due to
		// not having a tracking branch.
		logger.Printf("No remote tracking branch for %s; done", newName)
		return nil
	}
	// Now we have an upstream ref like "origin/somebranch".
	var remote, remoteBranch string
	switch strings.Count(upstream, "/") {
	case 0:
		return fmt.Errorf(
			"upstream %q returned by 'git rev-parse' has no remote name",
			upstream,
		)
	case 1:
		remote, remoteBranch, _ = strings.Cut(upstream, "/")
	default:
		// Unfortunately, both the name of the remote and the branch can
		// have slashes in them, so an upstream like "foo/bar/baz" is
		// ambiguous.
		logger.Printf("Disambiguating remote vs. branch in upstream %q", upstream)
		var err error
		configRemote := fmt.Sprintf("branch.%s.remote", newName)
		remote, err = runGit(logger, "config", configRemote)
		if err != nil {
			return err
		}
		var ok bool
		remoteBranch, ok = strings.CutPrefix(upstream, remote+"/")
		if !ok {
			return fmt.Errorf(
				"upstream %q does not begin with name of remote %q",
				upstream, remote,
			)
		}
	}
	if remoteBranch != oldName {
		return fmt.Errorf(
			"renamed %s to %s, but its remote tracking branch has a different name (%s)",
			oldName, newName, remoteBranch,
		)
	}
	_, err = runGit(logger, "push", remote, ":"+oldName, newName)
	if err != nil {
		return err
	}
	_, err = runGit(logger, "push", "-u", remote, "-u", newName+":"+newName)
	if err != nil {
		return err
	}
	return nil
}

func currentBranch(logger *log.Logger) (string, error) {
	branch, err := runGit(logger, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		// The man page says that symbolic-ref returns status 1 if the
		// requested name is not a symbolic ref and 128 for other errors,
		// but in practice I get status 128 if the name is not a
		// symbolic ref. So just look for the error string for now.
		// TODO: figure out if this is a bug; perhaps it's fixed in a
		// later version.
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if bytes.Contains(ee.Stderr, []byte("not a symbolic ref")) {
				return "", errors.New("not on a branch (detached HEAD)?")
			}
		}
		return "", err
	}
	return branch, nil
}

func runGit(logger *log.Logger, cmd string, args ...string) (string, error) {
	fullCmd := slices.Concat([]string{"git", cmd}, args)
	logger.Printf("> %s", strings.Join(fullCmd, " "))
	out, err := exec.Command("git", fullCmd[1:]...).Output()
	if err != nil {
		return "", &gitError{
			fullCmd: fullCmd,
			err:     err,
		}
	}
	return strings.TrimSpace(string(out)), nil
}

type gitError struct {
	fullCmd []string
	err     error
}

func (e *gitError) Unwrap() error { return e.err }

func (e *gitError) Error() string {
	msg := fmt.Sprintf("exec %q: %s", strings.Join(e.fullCmd, " "), e.err)
	var ee *exec.ExitError
	if errors.As(e.err, &ee) && len(ee.Stderr) > 0 {
		msg += fmt.Sprintf("\nstderr:\n%s", ee.Stderr)
	}
	return msg
}
