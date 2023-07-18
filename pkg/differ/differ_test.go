package differ

import (
	"testing"

	"github.com/busser/tfautomv/pkg/differ/rules"
	"github.com/busser/tfautomv/pkg/terraform"
	"github.com/google/go-cmp/cmp"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name string

		create terraform.Resource
		delete terraform.Resource
		rules  []rules.Rule

		want terraform.Diff
	}{
		{
			name: "without rules",
			create: dummyResource(map[string]any{
				"a": "hello",
				"b": 123,
				"c": true,
				"d": nil,
				"e": "foo",
				"f": 456,
				"h": "goodbye",
			}),
			delete: dummyResource(map[string]any{
				"a": "hello",
				"b": 123,
				"c": false,
				"d": 12.34,
				"e": nil,
				"g": "whatever",
				"h": 789,
			}),
			want: terraform.Diff{
				MatchingAttributes:    []string{"a", "b"},
				MismatchingAttributes: []string{"c", "e", "f", "h"},
			},
		},

		{
			name: "with rules",
			create: dummyResource(map[string]any{
				"a": "hello",
				"b": 123,
				"c": true,
				"d": nil,
				"e": "foo",
				"f": 456,
				"h": "goodbye",
				"i": "{\"foo\":\"bar\"}",
				"j": "some_string",
			}),
			delete: dummyResource(map[string]any{
				"a": "hello",
				"b": 123,
				"c": false,
				"d": 12.34,
				"e": nil,
				"g": "whatever",
				"h": 789,
				"i": "{\n\t\"foo\": \"bar\"\n}",
				"j": "b/some_string",
			}),
			rules: []rules.Rule{
				rules.MustParse("everything:dummy_type:c"),
				rules.MustParse("whitespace:dummy_type:i"),
				rules.MustParse("prefix:dummy_type:j:b/"),
			},
			want: terraform.Diff{
				MatchingAttributes:    []string{"a", "b"},
				MismatchingAttributes: []string{"e", "f", "h"},
				IgnoredAttributes:     []string{"c", "i", "j"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newTestDiffer(t, tt.rules).Diff(tt.create, tt.delete)
			assertDiffsAreEqual(t, tt.want, got)
		})
	}

}

func dummyResource(attributes map[string]any) terraform.Resource {
	return terraform.Resource{
		Module: terraform.Module{
			Path: "dummy_module_path",
		},
		Type:       "dummy_type",
		Address:    "dummy_address",
		Attributes: attributes,
	}
}

func newTestDiffer(t *testing.T, rules []rules.Rule) *Differ {
	t.Helper()

	differ, err := New(WithRules(rules...))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return differ
}

func assertDiffsAreEqual(t *testing.T, want, got terraform.Diff) {
	t.Helper()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
