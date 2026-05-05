package index

import (
	"math"
	"testing"
)

func TestIDF(t *testing.T) {
	tests := []struct {
		name string
		N    int
		df   int
		// expected: roughly ln((N-df+0.5)/(df+0.5) + 1)
		wantPositive bool
		wantZero     bool
	}{
		{name: "df=0 returns 0", N: 100, df: 0, wantZero: true},
		{name: "rare term has high IDF", N: 1000, df: 1, wantPositive: true},
		{name: "common term has lower IDF", N: 1000, df: 500, wantPositive: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := idf(tt.N, tt.df)
			if tt.wantZero && got != 0 {
				t.Errorf("idf(%d,%d) = %f, want 0", tt.N, tt.df, got)
			}
			if tt.wantPositive && got <= 0 {
				t.Errorf("idf(%d,%d) = %f, want > 0", tt.N, tt.df, got)
			}
		})
	}

	// Rare terms should score higher than common terms.
	rarIDF := idf(1000, 1)
	comIDF := idf(1000, 500)
	if rarIDF <= comIDF {
		t.Errorf("rare IDF (%f) should exceed common IDF (%f)", rarIDF, comIDF)
	}
}

func TestIDFFormula(t *testing.T) {
	N, df := 10, 2
	expected := math.Log((float64(N-df)+0.5)/(float64(df)+0.5) + 1.0)
	got := idf(N, df)
	if math.Abs(got-expected) > 1e-9 {
		t.Errorf("idf formula mismatch: got %f want %f", got, expected)
	}
}

func TestSearch_Empty(t *testing.T) {
	idx := New()
	if hits := Search(idx, []string{"foo"}, 10); hits != nil {
		t.Errorf("expected nil hits on empty index, got %v", hits)
	}
	if hits := Search(idx, nil, 10); hits != nil {
		t.Errorf("expected nil hits for nil query, got %v", hits)
	}
}

func TestSearch_SingleDocument(t *testing.T) {
	idx := New()
	idx.AddDocument(1, []string{"search", "engine", "search", "fast"}, "search engine search fast")

	hits := Search(idx, []string{"search"}, 10)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].DocID != 1 {
		t.Errorf("expected docID 1, got %d", hits[0].DocID)
	}
	if hits[0].Score <= 0 {
		t.Errorf("expected positive score, got %f", hits[0].Score)
	}
}

func TestSearch_Ranking(t *testing.T) {
	idx := New()
	idx.AddDocument(1, []string{"search", "engine"}, "search engine")
	idx.AddDocument(2, []string{"search", "search", "engine"}, "search search engine")

	hits := Search(idx, []string{"search"}, 10)
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
	if hits[0].DocID != 2 {
		t.Errorf("expected doc 2 to rank first (higher tf), got doc %d", hits[0].DocID)
	}
}

func TestSearch_TopK(t *testing.T) {
	idx := New()
	for i := 1; i <= 10; i++ {
		idx.AddDocument(i, []string{"common", "term"}, "common term")
	}

	hits := Search(idx, []string{"common"}, 3)
	if len(hits) != 3 {
		t.Errorf("expected 3 hits (topK=3), got %d", len(hits))
	}
}

func TestSearch_MultiTermQuery(t *testing.T) {
	idx := New()
	idx.AddDocument(1, []string{"inverted", "index", "search"}, "inverted index search")
	idx.AddDocument(2, []string{"database", "index"}, "database index")

	hits := Search(idx, []string{"inverted", "index"}, 10)
	if len(hits) < 1 {
		t.Fatal("expected hits, got none")
	}
	if hits[0].DocID != 1 {
		t.Errorf("expected doc 1 to rank first, got doc %d", hits[0].DocID)
	}
}

func TestSearch_NoMatchingTerm(t *testing.T) {
	idx := New()
	idx.AddDocument(1, []string{"hello", "world"}, "hello world")

	hits := Search(idx, []string{"nosuchterm"}, 10)
	if len(hits) != 0 {
		t.Errorf("expected 0 hits, got %d", len(hits))
	}
}

func TestSearch_SnippetIncluded(t *testing.T) {
	idx := New()
	idx.AddDocument(1, []string{"search", "engine"}, "search engine document")

	hits := Search(idx, []string{"search"}, 10)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Snippet == "" {
		t.Error("expected non-empty snippet in search result")
	}
}

func TestIndex_AddRemove(t *testing.T) {
	idx := New()
	idx.AddDocument(1, []string{"foo", "bar", "foo"}, "foo bar foo")
	idx.AddDocument(2, []string{"bar", "baz"}, "bar baz")

	totalDocs, _ := idx.Snapshot()
	if totalDocs != 2 {
		t.Errorf("expected TotalDocs=2, got %d", totalDocs)
	}

	idx.RemoveDocument(1)
	totalDocs, _ = idx.Snapshot()
	if totalDocs != 1 {
		t.Errorf("expected TotalDocs=1 after remove, got %d", totalDocs)
	}

	postings := idx.GetPostings("foo")
	if len(postings) != 0 {
		t.Errorf("expected no postings for 'foo' after remove, got %v", postings)
	}
}

func TestIndex_RemoveNonExistent(t *testing.T) {
	idx := New()
	idx.RemoveDocument(999)
}
