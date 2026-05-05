package index

import "sync"

// PostingEntry records how often a term appears in a document.
type PostingEntry struct {
	DocID    int
	TermFreq int
}

// Index is the in-memory inverted index.
// All exported methods are safe for concurrent use.
type Index struct {
	mu         sync.RWMutex
	Postings   map[string][]PostingEntry // term -> posting list
	DocLengths map[int]int               // doc_id -> token count
	AvgDocLen  float64
	TotalDocs  int
}

// New returns an empty, ready-to-use Index.
func New() *Index {
	return &Index{
		Postings:   make(map[string][]PostingEntry),
		DocLengths: make(map[int]int),
	}
}

// AddDocument indexes a document given its ID and pre-tokenized terms.
// Calling AddDocument with a docID that already exists will double-count it;
// callers must ensure each docID is added at most once.
func (idx *Index) AddDocument(docID int, tokens []string) {
	// Compute per-term frequencies.
	tf := make(map[string]int, len(tokens))
	for _, t := range tokens {
		tf[t]++
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	for term, freq := range tf {
		idx.Postings[term] = append(idx.Postings[term], PostingEntry{
			DocID:    docID,
			TermFreq: freq,
		})
	}

	docLen := len(tokens)
	idx.DocLengths[docID] = docLen
	idx.TotalDocs++

	// Recompute average doc length incrementally.
	// avgNew = avgOld + (docLen - avgOld) / totalDocs
	idx.AvgDocLen += (float64(docLen) - idx.AvgDocLen) / float64(idx.TotalDocs)
}

// RemoveDocument removes a document and all its postings from the index.
// If the docID does not exist, RemoveDocument is a no-op.
func (idx *Index) RemoveDocument(docID int) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	docLen, ok := idx.DocLengths[docID]
	if !ok {
		return
	}

	// Remove from posting lists.
	for term, postings := range idx.Postings {
		filtered := postings[:0]
		for _, p := range postings {
			if p.DocID != docID {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			delete(idx.Postings, term)
		} else {
			idx.Postings[term] = filtered
		}
	}

	delete(idx.DocLengths, docID)
	idx.TotalDocs--

	// Recompute average doc length.
	if idx.TotalDocs == 0 {
		idx.AvgDocLen = 0
	} else {
		// Reverse the incremental formula:
		// avgOld * (n+1) = avgNew * n + docLen
		// => avgNew = (avgOld*(n+1) - docLen) / n
		idx.AvgDocLen = (idx.AvgDocLen*float64(idx.TotalDocs+1) - float64(docLen)) / float64(idx.TotalDocs)
	}
}

// GetPostings returns the posting list for a term (nil if not found).
// The returned slice must not be modified by the caller.
func (idx *Index) GetPostings(term string) []PostingEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.Postings[term]
}

// DocFreq returns the number of documents containing the given term.
func (idx *Index) DocFreq(term string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.Postings[term])
}

// DocLength returns the token count for a document, and whether it exists.
func (idx *Index) DocLength(docID int) (int, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	l, ok := idx.DocLengths[docID]
	return l, ok
}

// Snapshot returns a consistent read of the index statistics needed for BM25.
func (idx *Index) Snapshot() (totalDocs int, avgDocLen float64) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.TotalDocs, idx.AvgDocLen
}

// TermCount returns the number of unique terms in the index.
func (idx *Index) TermCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.Postings)
}
