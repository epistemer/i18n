package labels

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/epistemer/genx/genxtest"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// i18nDir is where the generator lives: i18n's root package directory.
const i18nDir = ".."

// render runs the labels generator on the JSON files of a testdata
// directory, in name order.
func render(t testing.TB, dir string) map[string][]byte {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", dir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no input in testdata/%s: %v", dir, err)
	}
	sort.Strings(files)
	var in []genxtest.Input
	for _, f := range files {
		in = append(in, genxtest.Input{Path: f})
	}
	return genxtest.Render(t, i18nDir, "labels", in...)
}

// declarations drops comments and blank lines, which carry no declaration.
func declarations(b []byte) string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		if s := strings.TrimSpace(line); s == "" || strings.HasPrefix(s, "//") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// TestCfgxZP is M6.1's done-when: the labels of cfgx's zp example render to
// the same declarations as the zz_settings_i18n.go that cfgx's App.I18n()
// wrote (cfgx commit 5a1759c); only comments differ.
func TestCfgxZP(t *testing.T) {
	files := genxtest.Render(t, i18nDir, "labels", genxtest.Input{Path: "testdata/zp.labels.json"})
	got := files["internal/settings/zz_settings_i18n.go"]
	want, err := os.ReadFile("testdata/cfgx_zz_settings_i18n.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	if g, w := declarations(got), declarations(want); g != w {
		t.Errorf("declarations differ from cfgx's file\ngot:\n%s\nwant:\n%s", g, w)
	}
	if n := strings.Count(declarations(got), "\n") + 1; n != 117 {
		t.Errorf("%d lines of declarations, want 117", n)
	}
}

// TestGolden renders each directory of testdata/golden and compares every
// file written with want/<path>. A case with no want directory writes nothing.
func TestGolden(t *testing.T) {
	cases, err := filepath.Glob(filepath.Join("testdata", "golden", "*"))
	if err != nil || len(cases) == 0 {
		t.Fatalf("no golden cases: %v", err)
	}
	for _, c := range cases {
		name := filepath.Base(c)
		t.Run(name, func(t *testing.T) {
			files := render(t, filepath.Join("golden", name))
			wantDir := filepath.Join(c, "want")
			if *update {
				os.RemoveAll(wantDir)
				for p, b := range files {
					dst := filepath.Join(wantDir, filepath.FromSlash(p)+".golden")
					if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(dst, b, 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			want := map[string][]byte{}
			filepath.WalkDir(wantDir, func(p string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					rel, _ := filepath.Rel(wantDir, p)
					b, _ := os.ReadFile(p)
					want[strings.TrimSuffix(filepath.ToSlash(rel), ".golden")] = b
				}
				return nil
			})
			for p, b := range want {
				if !bytes.Equal(files[p], b) {
					t.Errorf("%s:\ngot:\n%s\nwant:\n%s", p, files[p], b)
				}
			}
			for p := range files {
				if _, ok := want[p]; !ok {
					t.Errorf("wrote %s, which the case doesn't expect", p)
				}
			}
		})
	}
}

// failure records the message genxtest.Render fails with, so a test can
// check what the generator reports.
type failure struct {
	testing.TB
	msg string
}

func (f *failure) Helper() {}

func (f *failure) Fatalf(format string, args ...any) {
	f.msg = fmt.Sprintf(format, args...)
}

// TestChecks is M6.2: each check's message names the key or identifier and
// the producers involved.
func TestChecks(t *testing.T) {
	tests := []struct {
		dir, want string
	}{
		{"twotexts", "labels: one key, two texts: settings.title (enums, settings)"},
		{"twovars", "labels: declared twice in package internal/settings: var L10nTitle (enums, settings)"},
		{"twopaths", "labels: declared twice in package internal/settings: path title (enums, settings)"},
		{"twonames", "labels: package internal/settings is named config and settings (enums, settings)"},
	}
	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			f := &failure{TB: t}
			files := render(f, filepath.Join("checks", tt.dir))
			if files != nil || f.msg == "" {
				t.Fatalf("rendered %d files, want a failure", len(files))
			}
			if !strings.Contains(f.msg, tt.want) {
				t.Errorf("message:\n%s\nwant it to contain:\n%s", f.msg, tt.want)
			}
		})
	}
}
