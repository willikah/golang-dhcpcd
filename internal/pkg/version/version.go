package version

import _ "embed"

//go:generate sh -c "printf %s $(git rev-parse HEAD) > commit.txt"
//go:generate sh -c "printf %s $(git rev-parse --abbrev-ref HEAD) > branch.txt"
//go:generate sh -c "printf %s $(git describe --tags --abbrev=0 2>/dev/null || echo none) > tag.txt"
//go:generate sh -c "git diff-index --quiet HEAD -- || echo dirty > dirty.txt; [ -f dirty.txt ] || echo clean > dirty.txt"

//go:embed commit.txt
var commit string

//go:embed branch.txt
var branch string

//go:embed tag.txt
var tag string

//go:embed dirty.txt
var dirty string

type gitInfo struct {
	Commit string
	Branch string
	Tag    string
	Dirty  bool
}

var info = gitInfo{
	Commit: commit,
	Branch: branch,
	Tag:    tag,
	Dirty:  dirty == "dirty",
}

// GetGitInfo returns a copy of the gitInfo struct containing git metadata.
func GetGitInfo() gitInfo {
	return info
}
