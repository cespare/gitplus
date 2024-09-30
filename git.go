package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"slices"
	"strconv"
	"strings"
)

// aheadBehind reports the number of commits that are on ref0 and not on ref1
// (ahead) and how many commits are on ref1 but not ref0 (behind).
func aheadBehind(logger *log.Logger, ref0, ref1 string) (ahead, behind int, err error) {
	out, err := runGit(logger, "rev-list", "--left-right", "--count", ref0, ref1)
	if err != nil {
		return 0, 0, err
	}
	left, right, ok := strings.Cut(out, "\t")
	if !ok {
		return 0, 0, fmt.Errorf("unexpected rev-list --left-right --count output: %q", out)
	}
	ahead, err = strconv.Atoi(left)
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing rev-list output: %q is not an int", left)
	}
	behind, err = strconv.Atoi(right)
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing rev-list output: %q is not an int", right)
	}
	return ahead, behind, nil
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
