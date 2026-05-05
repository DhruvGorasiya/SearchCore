package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/dhruvgorasiya/searchcore/internal/index"
	"github.com/dhruvgorasiya/searchcore/internal/service"
	"github.com/dhruvgorasiya/searchcore/internal/store"
	pb "github.com/dhruvgorasiya/searchcore/pb"
)

const migrationSQL = `
CREATE TABLE IF NOT EXISTS documents (
    id          SERIAL PRIMARY KEY,
    content     TEXT        NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS postings (
    term        TEXT    NOT NULL,
    doc_id      INT     NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    term_freq   INT     NOT NULL,
    PRIMARY KEY (term, doc_id)
);

CREATE TABLE IF NOT EXISTS corpus_stats (
    key         TEXT PRIMARY KEY,
    value       FLOAT NOT NULL
);

CREATE TABLE IF NOT EXISTS term_stats (
    term        TEXT PRIMARY KEY,
    doc_freq    INT NOT NULL
);

INSERT INTO corpus_stats (key, value) VALUES ('total_docs', 0), ('avg_doc_length', 0)
ON CONFLICT (key) DO NOTHING;
`

func main() {
	dsn := getenv("DATABASE_URL", "postgres://searchcore:searchcore@localhost:5432/searchcore?sslmode=disable")
	addr := getenv("GRPC_ADDR", ":50051")

	// 1. Connect to PostgreSQL.
	st, err := store.NewStore(dsn)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer st.Close()
	log.Println("connected to database")

	// 2. Run migrations.
	if err := st.RunMigrations(migrationSQL); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	log.Println("migrations applied")

	// 3. Reconstruct in-memory index from DB.
	idx, err := loadIndex(st)
	if err != nil {
		log.Fatalf("load index: %v", err)
	}
	totalDocs, avgLen := idx.Snapshot()
	log.Printf("index loaded: %d documents, %d terms, avg_len=%.1f", totalDocs, idx.TermCount(), avgLen)

	// 4. Start gRPC server.
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen on %s: %v", addr, err)
	}

	grpcServer := grpc.NewServer()
	svc := service.New(idx, st)
	pb.RegisterSearchServiceServer(grpcServer, svc)

	log.Printf("gRPC server listening on %s", addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// loadIndex reconstructs the in-memory index from the persisted postings and stats.
func loadIndex(st *store.Store) (*index.Index, error) {
	postings, err := st.LoadAllPostings()
	if err != nil {
		return nil, fmt.Errorf("load postings: %w", err)
	}

	corpusStats, err := st.LoadCorpusStats()
	if err != nil {
		return nil, fmt.Errorf("load corpus stats: %w", err)
	}

	// Group postings by doc_id to reconstruct DocLengths.
	// We compute doc lengths by summing term frequencies per document.
	docTF := make(map[int]map[string]int) // docID -> term -> freq
	for _, p := range postings {
		if docTF[p.DocID] == nil {
			docTF[p.DocID] = make(map[string]int)
		}
		docTF[p.DocID][p.Term] = p.TermFreq
	}

	idx := index.New()

	// Rebuild posting lists and doc lengths directly (no per-doc avg recalculation).
	for docID, terms := range docTF {
		var docLen int
		entries := make(map[string][]index.PostingEntry)
		for term, freq := range terms {
			entries[term] = append(entries[term], index.PostingEntry{DocID: docID, TermFreq: freq})
			docLen += freq
		}
		for term, e := range entries {
			for _, pe := range e {
				idx.SetPostings(term, append(idx.GetPostings(term), pe))
			}
		}
		idx.SetDocLength(docID, docLen)
	}

	// Restore corpus-level stats.
	idx.SetCorpusStats(int(corpusStats["total_docs"]), corpusStats["avg_doc_length"])

	// Pre-warm snippet cache.
	snippets, err := st.LoadAllDocumentSnippets()
	if err != nil {
		return nil, fmt.Errorf("load snippets: %w", err)
	}
	for docID, snippet := range snippets {
		idx.SetSnippet(docID, snippet)
	}

	return idx, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
