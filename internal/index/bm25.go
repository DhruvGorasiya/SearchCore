package index

import (
	"math"
	"sort"
)

const (
	k1 = 1.5
	b  = 0.75
)

// SearchHit is a ranked result from a BM25 search.
type SearchHit struct {
	DocID int
	Score float64
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
// query should already be tokenized by the same pipeline used at index time.
func Search(idx *Index, query []string, topK int) []SearchHit {
	if len(query) == 0 || topK <= 0 {
		return nil
	}

	totalDocs, avgDocLen := idx.Snapshot()
	if totalDocs == 0 || avgDocLen == 0 {
		return nil
	}

	scores := make(map[int]float64)

	for _, term := range query {
		postings := idx.GetPostings(term)
		if len(postings) == 0 {
			continue
		}
		df := len(postings)
		idfVal := idf(totalDocs, df)

		for _, p := range postings {
			docLen, ok := idx.DocLength(p.DocID)
			if !ok {
				continue
			}
			scores[p.DocID] += termScore(p.TermFreq, docLen, avgDocLen, idfVal)
		}
	}

	hits := make([]SearchHit, 0, len(scores))
	for docID, score := range scores {
		hits = append(hits, SearchHit{DocID: docID, Score: score})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].DocID < hits[j].DocID
	})

	if topK > len(hits) {
		topK = len(hits)
	}
	return hits[:topK]
}
