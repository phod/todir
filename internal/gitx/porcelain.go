package gitx

import "strings"

// Worktree describes one entry from `git worktree list --porcelain`.
type Worktree struct {
	Path, Branch, Head   string
	Detached, Bare, Main bool
}

// parsePorcelain parses the output of `git worktree list --porcelain`.
// The first entry is the main worktree.
func parsePorcelain(s string) ([]Worktree, error) {
	wts := []Worktree{}
	var cur *Worktree
	flush := func() {
		if cur == nil {
			return
		}
		if cur.Detached {
			cur.Branch = cur.Head
			if len(cur.Branch) > 7 {
				cur.Branch = cur.Branch[:7]
			}
		}
		cur.Main = len(wts) == 0
		wts = append(wts, *cur)
		cur = nil
	}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		key, val, _ := strings.Cut(line, " ")
		if key == "worktree" {
			flush()
			cur = &Worktree{Path: val}
			continue
		}
		if cur == nil {
			continue
		}
		switch key {
		case "HEAD":
			cur.Head = val
		case "branch":
			cur.Branch = strings.TrimPrefix(val, "refs/heads/")
		case "detached":
			cur.Detached = true
		case "bare":
			cur.Bare = true
		}
	}
	flush()
	return wts, nil
}
