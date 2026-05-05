package index

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "only stop words",
			input: "the a an is are",
			want:  []string{},
		},
		{
			name:  "lowercasing",
			input: "Hello World",
			want:  []string{"hello", "world"},
		},
		{
			name:  "punctuation stripped",
			input: "hello, world! foo.bar",
			want:  []string{"hello", "world", "foo", "bar"},
		},
		{
			name:  "mixed stop words and content",
			input: "the quick brown fox jumps over the lazy dog",
			want:  []string{"quick", "brown", "fox", "jumps", "lazy", "dog"},
		},
		{
			name:  "numbers preserved",
			input: "go1 version 123",
			want:  []string{"go1", "version", "123"},
		},
		{
			name:  "hyphenated words split",
			input: "state-of-the-art design",
			want:  []string{"state", "art", "design"},
		},
		{
			name:  "extra whitespace",
			input: "  hello   world  ",
			want:  []string{"hello", "world"},
		},
		{
			name:  "all punctuation",
			input: "!@#$%^&*()",
			want:  []string{},
		},
		{
			name:  "unicode letters preserved",
			input: "café naïve",
			want:  []string{"café", "naïve"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tokenize(tt.input)
			if len(got) == 0 && len(tt.want) == 0 {
				return // both empty — pass
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tokenize(%q)\n  got  %v\n  want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTokenize_StopWordConsistency(t *testing.T) {
	// Same query at index time and search time should produce identical tokens.
	query := "what is the best search engine for full text retrieval"
	a := Tokenize(query)
	b := Tokenize(query)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Tokenize is non-deterministic: %v vs %v", a, b)
	}
}
