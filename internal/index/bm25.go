package index

import (
	"container/heap"
	"math"
)

const (
	k1 = 1.5
	b  = 0.75
)

// SearchResult is a ranked result including the cached snippet.
type SearchResult struct {
	DocID   int
	Score   float64
	Snippet string
}

// idf computes the BM25 IDF for a term given corpus size N and document
// frequency df.
//
//	IDF(t) = ln((N - df + 0.5) / (df + 0.5) + 1)
func idf(N, df int) float64 {
	if df == 0 {
		return 0
	}
	return math.Log((float64(N-df)+0.5)/(float64(df)+0.5) + 1.0)
}

// termScore computes the BM25 contribution of a single term for a single document.
func termScore(tf, docLen int, avgDocLen float64, idfVal float64) float64 {
	tfF := float64(tf)
	lenNorm := 1 - b + b*float64(docLen)/avgDocLen
	return idfVal * (tfF * (k1 + 1)) / (tfF + k1*lenNorm)
}

// Search returns the top-k documents ranked by BM25 score for the given query tokens.
// It is lock-free: it atomically loads a snapshot of the index and scores against it.
func Search(idx *Index, query []string, topK int) []SearchResult {
	if len(query) == 0 || topK <= 0 {
		return nil
	}

	// Atomic load — no lock held during the entire computation.
	d := idx.load()

	if d.TotalDocs == 0 || d.AvgDocLen == 0 {
		return nil
	}

	scores := make(map[int]float64, 1024)

	for _, term := range query {
		postings := d.Postings[term]
		if len(postings) == 0 {
			continue
		}
		idfVal := idf(d.TotalDocs, len(postings))

		for _, p := range postings {
			scores[p.DocID] += termScore(p.TermFreq, d.DocLengths[p.DocID], d.AvgDocLen, idfVal)
		}
	}

	if len(scores) == 0 {
		return nil
	}

	// Use a min-heap of size topK for O(N log k) selection instead of full sort.
	h := &resultHeap{}
	heap.Init(h)

	for docID, score := range scores {
		if h.Len() < topK {
			heap.Push(h, SearchResult{DocID: docID, Score: score, Snippet: d.Snippets[docID]})
		} else if score > (*h)[0].Score {
			(*h)[0] = SearchResult{DocID: docID, Score: score, Snippet: d.Snippets[docID]}
			heap.Fix(h, 0)
		}
	}

	// Pop in reverse order to return highest-score first.
	results := make([]SearchResult, h.Len())
	for i := len(results) - 1; i >= 0; i-- {
		results[i] = heap.Pop(h).(SearchResult)
	}
	return results
}

// resultHeap is a min-heap of SearchResult by score (lowest score at root).
type resultHeap []SearchResult

func (h resultHeap) Len() int            { return len(h) }
func (h resultHeap) Less(i, j int) bool  { return h[i].Score < h[j].Score }
func (h resultHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *resultHeap) Push(x interface{}) { *h = append(*h, x.(SearchResult)) }
func (h *resultHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}
