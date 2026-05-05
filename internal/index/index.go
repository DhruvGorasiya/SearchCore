package index

import (
	"sync"
	"sync/atomic"
)

// PostingEntry records how often a term appears in a document.
type PostingEntry struct {
	DocID    int
	TermFreq int
}

// indexData holds the immutable snapshot of index state.
// Readers load this atomically; writers copy, mutate, then swap.
type indexData struct {
	Postings   map[string][]PostingEntry
	DocLengths map[int]int
	Snippets   map[int]string
	TotalDocs  int
	AvgDocLen  float64
}

// Index is the in-memory inverted index.
// Reads are lock-free via atomic pointer loads.
// Writes are serialized by a mutex and use copy-on-write semantics.
type Index struct {
	mu   sync.Mutex // serializes writes only
	data atomic.Pointer[indexData]
}

// New returns an empty, ready-to-use Index.
func New() *Index {
	idx := &Index{}
	idx.data.Store(&indexData{
		Postings:   make(map[string][]PostingEntry),
		DocLengths: make(map[int]int),
		Snippets:   make(map[int]string),
	})
	return idx
}

// load returns the current index snapshot. Safe for concurrent use; no lock required.
func (idx *Index) load() *indexData {
	return idx.data.Load()
}

// AddDocument indexes a document given its ID, pre-tokenized terms, and raw content.
func (idx *Index) AddDocument(docID int, tokens []string, content string) {
	tf := make(map[string]int, len(tokens))
	for _, t := range tokens {
		tf[t]++
	}
	docLen := len(tokens)

	snippet := content
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	old := idx.data.Load()
	d := copyData(old)

	for term, freq := range tf {
		d.Postings[term] = append(d.Postings[term], PostingEntry{DocID: docID, TermFreq: freq})
	}
	d.DocLengths[docID] = docLen
	d.Snippets[docID] = snippet
	d.TotalDocs++
	d.AvgDocLen += (float64(docLen) - d.AvgDocLen) / float64(d.TotalDocs)

	idx.data.Store(d)
}

// RemoveDocument removes a document from the index. No-op if docID doesn't exist.
func (idx *Index) RemoveDocument(docID int) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	old := idx.data.Load()
	docLen, ok := old.DocLengths[docID]
	if !ok {
		return
	}

	d := copyData(old)

	for term, postings := range d.Postings {
		filtered := postings[:0]
		for _, p := range postings {
			if p.DocID != docID {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			delete(d.Postings, term)
		} else {
			d.Postings[term] = filtered
		}
	}
	delete(d.DocLengths, docID)
	delete(d.Snippets, docID)

	d.TotalDocs--
	if d.TotalDocs == 0 {
		d.AvgDocLen = 0
	} else {
		d.AvgDocLen = (d.AvgDocLen*float64(d.TotalDocs+1) - float64(docLen)) / float64(d.TotalDocs)
	}

	idx.data.Store(d)
}

// Snapshot returns corpus-level statistics.
func (idx *Index) Snapshot() (totalDocs int, avgDocLen float64) {
	d := idx.load()
	return d.TotalDocs, d.AvgDocLen
}

// TermCount returns the number of unique indexed terms.
func (idx *Index) TermCount() int {
	return len(idx.load().Postings)
}

// GetPostings returns the posting list for a term (nil if not found).
func (idx *Index) GetPostings(term string) []PostingEntry {
	return idx.load().Postings[term]
}

// DocFreq returns the document frequency for a term.
func (idx *Index) DocFreq(term string) int {
	return len(idx.load().Postings[term])
}

// DocLength returns the token count for a document.
func (idx *Index) DocLength(docID int) (int, bool) {
	d := idx.load()
	l, ok := d.DocLengths[docID]
	return l, ok
}

// copyData creates a shallow copy of indexData with new top-level maps
// so writers can modify the copy without affecting concurrent readers.
func copyData(src *indexData) *indexData {
	dst := &indexData{
		TotalDocs: src.TotalDocs,
		AvgDocLen: src.AvgDocLen,
		Postings:  make(map[string][]PostingEntry, len(src.Postings)),
		DocLengths: make(map[int]int, len(src.DocLengths)),
		Snippets:  make(map[int]string, len(src.Snippets)),
	}
	for k, v := range src.Postings {
		dst.Postings[k] = v // slices are shared; append will copy-on-write
	}
	for k, v := range src.DocLengths {
		dst.DocLengths[k] = v
	}
	for k, v := range src.Snippets {
		dst.Snippets[k] = v
	}
	return dst
}

// --- Exported fields used directly by main.go loadIndex ---
// These are set once at startup before the server accepts requests, so no lock needed.

// SetPostings directly sets the posting list for a term (startup only).
func (idx *Index) SetPostings(term string, entries []PostingEntry) {
	d := idx.data.Load()
	d.Postings[term] = entries
}

// SetDocLength directly sets a document length (startup only).
func (idx *Index) SetDocLength(docID, length int) {
	d := idx.data.Load()
	d.DocLengths[docID] = length
}

// SetSnippet directly sets a document snippet (startup only).
func (idx *Index) SetSnippet(docID int, snippet string) {
	d := idx.data.Load()
	d.Snippets[docID] = snippet
}

// SetCorpusStats sets TotalDocs and AvgDocLen (startup only).
func (idx *Index) SetCorpusStats(totalDocs int, avgDocLen float64) {
	d := idx.data.Load()
	d.TotalDocs = totalDocs
	d.AvgDocLen = avgDocLen
}
