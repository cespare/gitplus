# gitplus

Gitplus is a collection of helper tools to augment git, making certain workflows
and common tasks easier.

## Installation

First, install the program:

    go install github.com/cespare/gitplus@latest

Now you can run `gitplus -h` to see a list of commands an invoke them as, for
example,

    gitplus rename-branch args...

Another way to use the tool is to make symlinks in your `$PATH` that point at
gitplus. If gitplus is invoked with the name `git-foo`, then it runs the `foo`
command. For example, you might have a `git-rename-branch` symlink that points
at gitplus. Then you can run

    git rename-branch args...

instead.

## `git rename-branch`

You can rename a local branch with `git branch -m`, but if it has a tracking
branch you usually want to rename that too. `git rename-branch` renames a local
branch as well as the remote tracking branch (if it has the same name).
