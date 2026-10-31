package gitx

import "testing"

const fixture = `worktree /home/u/proj
HEAD 1111111aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
branch refs/heads/main

worktree /home/u/proj-feature
HEAD 2222222bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
branch refs/heads/feature
locked because reasons

worktree /home/u/proj-det
HEAD abc1234def5678abc1234def5678abc1234def56
detached
prunable gitdir file points to non-existent location

worktree /srv/bare.git
bare
`

func TestParsePorcelain(t *testing.T) {
	for _, in := range []string{fixture, fixture[:len(fixture)-1]} {
		wts, err := parsePorcelain(in)
		if err != nil {
			t.Fatal(err)
		}
		if len(wts) != 4 {
			t.Fatalf("len = %d", len(wts))
		}
		if !wts[0].Main || wts[1].Main || wts[0].Branch != "main" || wts[0].Path != "/home/u/proj" {
			t.Errorf("bad main/linked: %+v %+v", wts[0], wts[1])
		}
		if wts[1].Branch != "feature" {
			t.Errorf("branch = %q", wts[1].Branch)
		}
		if !wts[2].Detached || wts[2].Branch != "abc1234" {
			t.Errorf("detached: %+v", wts[2])
		}
		if !wts[3].Bare || wts[3].Branch != "" {
			t.Errorf("bare: %+v", wts[3])
		}
	}
}

func TestParsePorcelainEmpty(t *testing.T) {
	wts, err := parsePorcelain("")
	if err != nil || len(wts) != 0 {
		t.Fatalf("got %v, %v", wts, err)
	}
}

func TestDisplayName(t *testing.T) {
	if got := DisplayName("/a/monorepo/.git"); got != "monorepo" {
		t.Errorf("got %q", got)
	}
	if got := DisplayName("/srv/repo.git"); got != "repo.git" {
		t.Errorf("got %q", got)
	}
}
