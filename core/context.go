package core

type GitContext struct {
	Branch         string
	Target         string
	CommitDistance int
	ShortSHA       string
	HasRelease     bool
	ReleaseMajor   int
	ReleaseMinor   int
	VersionerMerge bool
}

func (g GitContext) IsBranch(prefix string) bool {
	return len(g.Branch) >= len(prefix) && g.Branch[:len(prefix)] == prefix
}
