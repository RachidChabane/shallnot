package proof_test

import (
	"testing"

	"github.com/RachidChabane/shallnot/internal/domain/proof"
)

func TestScopeContains(t *testing.T) {
	scope := proof.Scope{
		Code:    []string{"src", "lib/**/*.go", "tools/gen.py"},
		Tests:   []string{"tests", "e2e/*_test.py"},
		Exclude: []string{"**/node_modules/**", "src/generated.py"},
	}
	cases := []struct {
		role proof.InputRole
		file string
		want bool
	}{
		{proof.RoleCodeTree, "src/a.py", true},
		{proof.RoleCodeTree, "src/pkg/deep/b.py", true},
		{proof.RoleCodeTree, "lib/x/y.go", true},
		{proof.RoleCodeTree, "tools/gen.py", true},
		{proof.RoleCodeTree, "tools/other.py", false},
		{proof.RoleCodeTree, "srcs/a.py", false},
		{proof.RoleCodeTree, "tests/a_test.py", false},
		{proof.RoleCodeTree, "src/generated.py", false},
		{proof.RoleCodeTree, "src/node_modules/dep/x.js", false},
		{proof.RoleCodeTree, "src/../secret.py", false},
		{proof.RoleCodeTree, "/abs/src/a.py", false},
		{proof.RoleTestSource, "tests/a_test.py", true},
		{proof.RoleTestSource, "e2e/flow_test.py", true},
		{proof.RoleTestSource, "e2e/flow.py", false},
		{proof.RoleTestSource, "src/a.py", false},
		{proof.RoleModelID, "src/a.py", false},
	}
	for _, c := range cases {
		t.Run(string(c.role)+" "+c.file+" [verifies SN-133~2]", func(t *testing.T) {
			if got := scope.Contains(c.role, c.file); got != c.want {
				t.Fatalf("Contains = %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsRelativePOSIXPath(t *testing.T) {
	t.Run("refuses a path that is not relative POSIX [verifies SN-136~1]", func(t *testing.T) {
		for _, path := range []string{"/abs.txt", "../up.txt", "a//b.txt", "a/./b.txt", `a\\b.txt`, ""} {
			if proof.IsRelativePOSIXPath(path) {
				t.Errorf("accepted %q", path)
			}
		}
		if !proof.IsRelativePOSIXPath("a/b.txt") {
			t.Error("refused a/b.txt")
		}
	})
}
