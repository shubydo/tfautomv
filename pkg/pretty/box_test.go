package pretty

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var update bool

func init() {
	flag.BoolVar(&update, "update", false, "update golden files")
}

func MainTest(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestBoxItems(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		color string
	}{
		{
			name:  "single item",
			items: []string{"lorem ipsum"},
			color: "red",
		},
		{
			name:  "multiple items",
			items: []string{"lorem ipsum", "lorem ipsum"},
			color: "green",
		},
		{
			name:  "multiline items",
			items: []string{"lorem ipsum\nlorem ipsum", "lorem ipsum\nlorem ipsum"},
			color: "yellow",
		},
		{
			name:  "multiline items with empty lines",
			items: []string{"lorem ipsum\n\nlorem ipsum", "lorem ipsum\n\nlorem ipsum"},
			color: "magenta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BoxItems(tt.items, tt.color)

			// Compare the output to the golden file, and update it if necessary
			// and the user has specified the -update flag.
			goldenFile := fmt.Sprintf("testdata/%s.golden.txt", t.Name())
			wantBytes, err := ioutil.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("failed to read golden file: %v", err)
			}
			want := string(wantBytes)

			if diff := cmp.Diff(want, got); diff != "" {
				if update {
					t.Logf("updating golden file for test case %q", t.Name())
					if err := os.WriteFile(goldenFile, []byte(got), 0644); err != nil {
						t.Fatalf("failed to update golden file: %v", err)
					}
				} else {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestBoxSection(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		content string
		color   string
	}{
		{
			name:    "without title",
			title:   "",
			content: "lorem ipsum",
			color:   "red",
		},
		{
			name:    "with title",
			title:   "title",
			content: "lorem ipsum",
			color:   "blue",
		},
		{
			name:    "multiline content",
			title:   "title",
			content: "lorem ipsum\nlorem ipsum\nlorem ipsum",
			color:   "green",
		},
		{
			name:    "multiline content with empty lines",
			title:   "title",
			content: "lorem ipsum\n\nlorem ipsum\n\nlorem ipsum",
			color:   "yellow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BoxSection(tt.title, tt.content, tt.color)

			// Compare the output to the golden file, and update it if necessary
			// and the user has specified the -update flag.
			goldenFile := fmt.Sprintf("testdata/%s.golden.txt", t.Name())
			wantBytes, err := ioutil.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("failed to read golden file: %v", err)
			}
			want := string(wantBytes)

			if diff := cmp.Diff(want, got); diff != "" {
				if update {
					t.Logf("updating golden file for test case %q", t.Name())
					if err := os.WriteFile(goldenFile, []byte(got), 0644); err != nil {
						t.Fatalf("failed to update golden file: %v", err)
					}
				} else {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
