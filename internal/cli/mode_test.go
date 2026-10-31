package cli

import (
	"errors"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
		args  []string
		want  Mode
		usage bool
	}{
		{"no args picks alias", Flags{}, nil, Mode{Kind: KindPickAlias}, false},
		{"one arg navigates", Flags{}, []string{"foo"}, Mode{Kind: KindNavigate, Alias: "foo"}, false},
		{"two args usage", Flags{}, []string{"a", "b"}, Mode{}, true},
		{"register", Flags{Register: true}, []string{".", "foo"}, Mode{Kind: KindRegister, Path: ".", Alias: "foo"}, false},
		{"register worktree force", Flags{Register: true, Worktree: true, Force: true}, []string{".", "foo"}, Mode{Kind: KindRegister, Path: ".", Alias: "foo", Worktree: true, Force: true}, false},
		{"register one arg usage", Flags{Register: true}, []string{"foo"}, Mode{}, true},
		{"register list usage", Flags{Register: true, List: true}, []string{".", "foo"}, Mode{}, true},
		{"delete", Flags{Delete: true}, []string{"foo"}, Mode{Kind: KindDelete, Alias: "foo"}, false},
		{"delete no args usage", Flags{Delete: true}, nil, Mode{}, true},
		{"delete force usage", Flags{Delete: true, Force: true}, []string{"foo"}, Mode{}, true},
		{"list", Flags{List: true}, nil, Mode{Kind: KindListAliases}, false},
		{"list interactive", Flags{List: true, Interactive: true}, nil, Mode{Kind: KindPickAlias}, false},
		{"interactive", Flags{Interactive: true}, nil, Mode{Kind: KindPickAlias}, false},
		{"list with arg usage", Flags{List: true}, []string{"foo"}, Mode{}, true},
		{"worktree list", Flags{Worktree: true, List: true}, nil, Mode{Kind: KindListWorktrees}, false},
		{"worktree list alias", Flags{Worktree: true, List: true}, []string{"repo"}, Mode{Kind: KindListWorktrees, Alias: "repo"}, false},
		{"worktree list interactive alias", Flags{Worktree: true, List: true, Interactive: true}, []string{"repo"}, Mode{Kind: KindPickWorktree, Alias: "repo"}, false},
		{"worktree interactive", Flags{Worktree: true, Interactive: true}, nil, Mode{Kind: KindPickWorktree}, false},
		{"worktree list two args usage", Flags{Worktree: true, List: true}, []string{"a", "b"}, Mode{}, true},
		{"worktree alone usage", Flags{Worktree: true}, nil, Mode{}, true},
		{"force alone usage", Flags{Force: true}, nil, Mode{}, true},
		{"init", Flags{Init: "zsh"}, nil, Mode{Kind: KindInit, Shell: "zsh"}, false},
		{"init with arg usage", Flags{Init: "zsh"}, []string{"x"}, Mode{}, true},
		{"init list usage", Flags{Init: "zsh", List: true}, nil, Mode{}, true},
		{"version", Flags{Version: true}, nil, Mode{Kind: KindVersion}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.flags, tt.args)
			if tt.usage {
				var ue *UsageError
				if !errors.As(err, &ue) {
					t.Fatalf("want *UsageError, got %v (mode %+v)", err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
