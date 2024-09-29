# gitplus

Gitplus is a collection of helper tools to augment git, making certain workflows
and common tasks easier.

## `git rename-branch`

You can rename a local branch with `git branch -m`, but if it has a tracking
branch you usually want to rename that too. `git rename-branch` renames a local
branch as well as the remote tracking branch (if it has the same name).
