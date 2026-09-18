// Package architecture_test checks structural rules of the code base.
package architecture_test

import (
	"os/exec"
	"strings"
	"testing"
)

const modulePath = "github.com/RachidChabane/shallnot"

func dependencies(t *testing.T, target string) []string {
	t.Helper()
	output, err := exec.Command("go", "list", "-deps", target).Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	return strings.Fields(string(output))
}

func imports(t *testing.T, target string) []string {
	t.Helper()
	output, err := exec.Command("go", "list", "-f", `{{join .Imports "\n"}}`, target).Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	return strings.Fields(string(output))
}

func TestCore(t *testing.T) {
	t.Run("the binary links no networking package [verifies SN-66~1]", func(t *testing.T) {
		for _, dependency := range dependencies(t, modulePath+"/cmd/shallnot") {
			if dependency == "net" || strings.HasPrefix(dependency, "net/") || strings.HasPrefix(dependency, "golang.org/x/net") {
				t.Errorf("the binary depends on %s", dependency)
			}
		}
	})

	t.Run("the domain and the analysis depend on no adapter and import no file access [verifies SN-67~1]", func(t *testing.T) {
		for _, core := range []string{"/internal/domain", "/internal/analysis"} {
			for _, dependency := range imports(t, modulePath+core) {
				if strings.HasPrefix(dependency, modulePath+"/internal/adapters") || dependency == "os" || dependency == "io/fs" {
					t.Errorf("%s depends on %s", core, dependency)
				}
			}
		}
	})
}
