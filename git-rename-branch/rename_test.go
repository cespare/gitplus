package main

import (
	"bufio"
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
)

func TestBasic(t *testing.T) {
	td := newTestDir(t)
	repo := td.makeRepo("repo1")
	_ = td.makeRepo("repo2")

	repo.git("checkout", "-b", "branch1")

	logger := testLogger()
	repo.chdir()

	if err := rename(logger, "", "branch2"); err != nil {
		t.Fatal(err)
	}
	got := repo.summarizeBranches("origin")
	want := [][2]string{
		{"branch2*", ""},
		{"main", ""},
	}
	if diff := gocmp.Diff(got, want); diff != "" {
		t.Fatalf("unexpected branch state (-got, +want):\n%s", diff)
	}

	repo.git("remote", "add", "origin", "../repo2")
	repo.git("push", "-u", "origin", "branch2")

	if err := rename(logger, "", "branch3"); err != nil {
		t.Fatal(err)
	}
	got = repo.summarizeBranches("origin")
	want = [][2]string{
		{"branch3*", "branch3"},
		{"main", ""},
	}
	if diff := gocmp.Diff(got, want); diff != "" {
		t.Fatalf("unexpected branch state (-got, +want):\n%s", diff)
	}

	if err := rename(logger, "branch3", "branch4"); err != nil {
		t.Fatal(err)
	}
	got = repo.summarizeBranches("origin")
	want = [][2]string{
		{"branch4*", "branch4"},
		{"main", ""},
	}
	if diff := gocmp.Diff(got, want); diff != "" {
		t.Fatalf("unexpected branch state (-got, +want):\n%s", diff)
	}

	repo.git("checkout", "-b", "branch5")
	if err := rename(logger, "branch4", "branch6"); err != nil {
		t.Fatal(err)
	}
	got = repo.summarizeBranches("origin")
	want = [][2]string{
		{"branch5*", ""},
		{"branch6", "branch6"},
		{"main", ""},
	}
	if diff := gocmp.Diff(got, want); diff != "" {
		t.Fatalf("unexpected branch state (-got, +want):\n%s", diff)
	}
}

func TestErrors(t *testing.T) {
	td := newTestDir(t)
	repo := td.makeRepo("repo1")
	_ = td.makeRepo("repo2")

	repo.git("checkout", "-b", "branch1")
	repo.git("remote", "add", "origin", "../repo2")
	repo.git("push", "-u", "origin", "branch1")

	logger := testLogger()
	repo.chdir()

	checkError := func(err error, want string) {
		t.Helper()
		if err == nil {
			t.Fatalf("got nil error; want substring %q", want)
		}
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("got error %q; want substring %q", err, want)
		}
	}

	checkError(rename(logger, "", "branch1"), "names are the same")

	td.writeFile(filepath.Join("repo1", "a.txt"), "a")
	repo.git("add", "a.txt")
	repo.git("commit", "-am", "Another commit")
	repo.git("checkout", "@^")
	checkError(rename(logger, "", "branch2"), "detached HEAD")

	repo.git("checkout", "-b", "branch2")
	repo.git("push", "-u", "origin", "branch2:differentname")
	checkError(
		rename(logger, "branch2", "branch3"),
		"tracking branch has a different name",
	)
	got := repo.summarizeBranches("origin")
	want := [][2]string{
		{"branch1", "branch1"},
		{"branch3*", "differentname"},
		{"main", ""},
	}
	if diff := gocmp.Diff(got, want); diff != "" {
		t.Fatalf("unexpected branch state (-got, +want):\n%s", diff)
	}
}

func TestSlashes(t *testing.T) {
	td := newTestDir(t)
	repo := td.makeRepo("repo1")
	_ = td.makeRepo("repo2")

	repo.git("checkout", "-b", "slashy/branch")
	repo.git("remote", "add", "foo/bar", "../repo2")
	repo.git("push", "-u", "foo/bar", "slashy/branch")

	logger := testLogger()
	repo.chdir()

	if err := rename(logger, "", "more/slashes"); err != nil {
		t.Fatal(err)
	}
	got := repo.summarizeBranches("foo/bar")
	want := [][2]string{
		{"main", ""},
		{"more/slashes*", "more/slashes"},
	}
	if diff := gocmp.Diff(got, want); diff != "" {
		t.Fatalf("unexpected branch state (-got, +want):\n%s", diff)
	}
}

func testLogger() *log.Logger {
	if testing.Verbose() {
		return log.New(os.Stderr, "", 0)
	} else {
		return log.New(io.Discard, "", 0)
	}
}

type testDir struct {
	t   *testing.T
	dir string
}

func newTestDir(t *testing.T) testDir {
	t.Helper()
	return testDir{
		t:   t,
		dir: t.TempDir(),
	}
}

func (td testDir) writeFile(name, content string) {
	td.t.Helper()
	name = filepath.Join(td.dir, name)
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		td.t.Fatal(err)
	}
}

func (td testDir) makeRepo(dir string) testRepo {
	td.t.Helper()
	td.exec("git", "init", "-b", "main", dir)
	tr := testRepo{
		t:   td.t,
		dir: filepath.Join(td.dir, dir),
	}
	td.writeFile(filepath.Join(dir, "README.txt"), "Hello")
	tr.git("add", "README.txt")
	tr.git("commit", "-am", "Initial commit")
	return tr
}

func (td testDir) exec(cmd string, args ...string) {
	td.t.Helper()
	c := exec.Command(cmd, args...)
	c.Dir = td.dir
	testExec(td.t, c)
}

type testRepo struct {
	t   *testing.T
	dir string
}

func (tr testRepo) chdir() {
	tr.t.Helper()
	prevDir, err := os.Getwd()
	if err != nil {
		tr.t.Fatal(err)
	}
	if err := os.Chdir(tr.dir); err != nil {
		tr.t.Fatal(err)
	}
	tr.t.Cleanup(func() {
		if err := os.Chdir(prevDir); err != nil {
			tr.t.Fatal(err)
		}
	})
}

func (tr testRepo) git(cmd string, args ...string) []byte {
	tr.t.Helper()
	gitArgs := append([]string{cmd}, args...)
	c := exec.Command("git", gitArgs...)
	// We could also use -C, but the command is nicer for printing in error
	// messages without a long file path inside.
	c.Dir = tr.dir
	return testExec(tr.t, c)
}

// summarizeBranches creates a list of local branches as well as remote
// branches. For each list element, b[0] is the local branch and b[1] is the
// remote branch (either may be empty). The current local branch is suffixed
// with "*".
//
// The list is sorted in the natural order.
func (tr testRepo) summarizeBranches(remoteName string) [][2]string {
	tr.t.Helper()
	out := tr.git(
		"for-each-ref",
		"--format", "%(refname:short)%(HEAD) %(upstream:short)",
		"refs/heads",
		"refs/remotes/"+remoteName,
	)
	var branches [][2]string
	tracked := make(map[string]struct{})
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		local, remote, ok := strings.Cut(line, " ")
		if ok {
			remote = strings.TrimSpace(remote)
			remote, ok = strings.CutPrefix(remote, remoteName+"/")
			if !ok {
				tr.t.Fatalf("branch %q not on remote %q", remote, remoteName)
			}
			tracked[remote] = struct{}{}
		} else if r, ok := strings.CutPrefix(local, remoteName+"/"); ok {
			local, remote = "", r
			if _, ok := tracked[remote]; ok {
				continue
			}
		}
		branches = append(branches, [2]string{local, remote})
	}
	if err := scanner.Err(); err != nil {
		tr.t.Fatal(err)
	}

	slices.SortFunc(branches, func(b0, b1 [2]string) int {
		if d := cmp.Compare(b0[0], b1[0]); d != 0 {
			return d
		}
		return cmp.Compare(b0[1], b1[1])
	})
	return branches
}

func testExec(t *testing.T, cmd *exec.Cmd) []byte {
	t.Helper()
	out, err := cmd.Output()
	if err != nil {
		msg := fmt.Sprintf("error running %s: %s", cmd, err)
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			msg += "\nstderr:\n" + string(ee.Stderr)
		}
		t.Fatal(msg)
	}
	return out
}
