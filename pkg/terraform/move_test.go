package terraform

import (
	"bytes"
	"flag"
	"fmt"
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

func TestWriteMovedBlocks(t *testing.T) {
	tests := []struct {
		name    string
		moves   []Move
		wantErr bool
	}{
		{
			name: "moves within same workdir",
			moves: []Move{
				{
					FromWorkdir: "/path/to/workdir",
					ToWorkdir:   "/path/to/workdir",
					FromAddress: "aws_instance.foo",
					ToAddress:   "aws_instance.bar",
				},
			},
		},
		{
			name: "moves between different workdirs",
			moves: []Move{
				{
					FromWorkdir: "/path/to/workdir1",
					ToWorkdir:   "/path/to/workdir2",
					FromAddress: "aws_instance.foo",
					ToAddress:   "aws_instance.bar",
				},
			},
			wantErr: true,
		},
		{
			name: "multiple moves within same workdir",
			moves: []Move{
				{
					FromWorkdir: "/path/to/workdir",
					ToWorkdir:   "/path/to/workdir",
					FromAddress: "aws_instance.foo",
					ToAddress:   "aws_instance.bar",
				},
				{
					FromWorkdir: "/path/to/workdir",
					ToWorkdir:   "/path/to/workdir",
					FromAddress: "aws_instance.baz",
					ToAddress:   "aws_instance.qux",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)

			err := WriteMovedBlocks(buf, tt.moves)

			// Check if the error is as expected
			if err != nil && !tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && tt.wantErr {
				t.Fatalf("expected error but got none")
			}

			// Compare the output to the golden file, and update it if necessary
			// and the user has specified the -update flag.
			goldenFile := fmt.Sprintf("testdata/%s.golden.tf", t.Name())
			wantBytes, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("failed to read golden file: %v", err)
			}
			want := string(wantBytes)

			got := buf.String()
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

func TestWriteMoveCommands(t *testing.T) {
	tests := []struct {
		name  string
		moves []Move
	}{
		{
			name: "moves within same workdir",
			moves: []Move{
				{
					FromWorkdir: "/path/to/workdir",
					ToWorkdir:   "/path/to/workdir",
					FromAddress: "aws_instance.foo",
					ToAddress:   "aws_instance.bar",
				},
			},
		},
		{
			name: "moves between different workdirs",
			moves: []Move{
				{
					FromWorkdir: "/path/to/workdir1",
					ToWorkdir:   "/path/to/workdir2",
					FromAddress: "aws_instance.foo",
					ToAddress:   "aws_instance.bar",
				},
			},
		},
		{
			name: "multiple moves within same workdir",
			moves: []Move{
				{
					FromWorkdir: "/path/to/workdir",
					ToWorkdir:   "/path/to/workdir",
					FromAddress: "aws_instance.foo",
					ToAddress:   "aws_instance.bar",
				},
				{
					FromWorkdir: "/path/to/workdir",
					ToWorkdir:   "/path/to/workdir",
					FromAddress: "aws_instance.baz",
					ToAddress:   "aws_instance.qux",
				},
			},
		},
		{
			name: "multiple moves between different workdirs",
			moves: []Move{
				{
					FromWorkdir: "/path/to/workdir1",
					ToWorkdir:   "/path/to/workdir2",
					FromAddress: "aws_instance.foo",
					ToAddress:   "aws_instance.bar",
				},
				{
					FromWorkdir: "/path/to/workdir3",
					ToWorkdir:   "/path/to/workdir4",
					FromAddress: "aws_instance.baz",
					ToAddress:   "aws_instance.qux",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)

			err := WriteMoveCommands(buf, tt.moves)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Compare the output to the golden file, and update it if necessary
			// and the user has specified the -update flag.
			goldenFile := fmt.Sprintf("testdata/%s.golden.sh", t.Name())
			wantBytes, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("failed to read golden file: %v", err)
			}
			want := string(wantBytes)

			got := buf.String()
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
