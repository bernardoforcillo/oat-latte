package oattest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	oat "github.com/antoniocali/oat-latte"
)

// RenderToString renders component c into a simulation screen of the given
// dimensions and returns an ASCII representation suitable for snapshot testing.
//
// Each terminal cell becomes a single rune in the output.
// Rows are separated by newlines. The result always ends with a newline.
func RenderToString(c oat.Component, width, height int) string {
	s := NewSimScreen(width, height)
	defer s.Fini()

	oat.RenderComponentToScreen(c, s.Screen())
	s.Show()

	var sb strings.Builder
	for y := 0; y < height; y++ {
		sb.WriteString(s.Row(y))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// AssertSnapshot renders c into a width×height snapshot and compares it against
// the golden file at goldenPath.
//
// First run (golden file does not exist): writes the rendered output to goldenPath
// and marks the test as skipped with a message explaining that the golden file was
// created and the test should be re-run.
//
// Subsequent runs: reads goldenPath, compares against the current render.
// On mismatch, calls t.Errorf with a unified diff showing the differences.
//
// Set the OATTEST_UPDATE=1 environment variable to overwrite existing golden files.
//
// Example:
//
//	func TestDashboard(t *testing.T) {
//	    d := myapp.BuildDashboard()
//	    oattest.AssertSnapshot(t, d, 80, 24, "testdata/dashboard.snap")
//	}
func AssertSnapshot(t testing.TB, c oat.Component, width, height int, goldenPath string) {
	t.Helper()

	got := RenderToString(c, width, height)
	update := os.Getenv("OATTEST_UPDATE") == "1"

	_, statErr := os.Stat(goldenPath)
	fileExists := statErr == nil

	if !fileExists || update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatalf("oattest.AssertSnapshot: failed to create directory for %s: %v", goldenPath, err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
			t.Fatalf("oattest.AssertSnapshot: failed to write golden file %s: %v", goldenPath, err)
		}
		if !fileExists {
			t.Skipf("oattest.AssertSnapshot: created snapshot %s — re-run to verify", goldenPath)
			return
		}
		// update == true and file already existed
		t.Logf("oattest.AssertSnapshot: updated snapshot %s", goldenPath)
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("oattest.AssertSnapshot: failed to read golden file %s: %v", goldenPath, err)
	}
	want := string(wantBytes)

	if want == got {
		return
	}

	t.Errorf("oattest.AssertSnapshot: snapshot mismatch for %s:\n%s", goldenPath, diff(want, got))
}

// MustMatchSnapshot is like AssertSnapshot but uses the test name and a
// "testdata/snapshots/" directory as the golden file path automatically.
//
//	oattest.MustMatchSnapshot(t, myWidget, 80, 24)
//	// golden file: testdata/snapshots/<TestName>.snap
func MustMatchSnapshot(t *testing.T, c oat.Component, width, height int) {
	t.Helper()
	goldenPath := filepath.Join("testdata", "snapshots", t.Name()+".snap")
	AssertSnapshot(t, c, width, height, goldenPath)
}

// diff returns a simple unified-diff-style string showing lines that differ
// between want and got. Lines present only in want are prefixed with "-",
// lines only in got are prefixed with "+".
func diff(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")

	var sb strings.Builder

	maxLen := len(wantLines)
	if len(gotLines) > maxLen {
		maxLen = len(gotLines)
	}

	for i := 0; i < maxLen; i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}

		if w == g {
			sb.WriteString("  ")
			sb.WriteString(w)
			sb.WriteByte('\n')
		} else {
			if i < len(wantLines) {
				sb.WriteString("- ")
				sb.WriteString(w)
				sb.WriteByte('\n')
			}
			if i < len(gotLines) {
				sb.WriteString("+ ")
				sb.WriteString(g)
				sb.WriteByte('\n')
			}
		}
	}

	return sb.String()
}
