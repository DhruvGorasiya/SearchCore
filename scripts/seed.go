//go:build ignore

// seed.go loads 10,000 documents into SearchCore via the gRPC API.
// Run with: go run scripts/seed.go
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/dhruvgorasiya/searchcore/pb"
)

// vocabulary is used to generate varied, realistic-looking documents.
var vocabulary = []string{
	"search", "engine", "index", "query", "document", "retrieval", "ranking",
	"relevance", "score", "term", "frequency", "inverted", "posting", "list",
	"bm25", "tfidf", "corpus", "token", "tokenize", "stemming", "lemmatize",
	"stop", "word", "filter", "pipeline", "precision", "recall", "latency",
	"throughput", "concurrent", "goroutine", "channel", "mutex", "lock",
	"database", "postgres", "transaction", "index", "schema", "table", "row",
	"grpc", "protobuf", "service", "server", "client", "request", "response",
	"golang", "performance", "benchmark", "profiling", "pprof", "memory",
	"allocation", "garbage", "collector", "heap", "stack", "pointer", "slice",
	"map", "struct", "interface", "method", "function", "package", "module",
	"data", "structure", "algorithm", "complexity", "logarithm", "linear",
	"binary", "tree", "graph", "hash", "bucket", "collision", "load", "factor",
	"distributed", "shard", "replica", "consensus", "raft", "leader", "follower",
	"network", "tcp", "http", "rest", "json", "serialization", "compression",
	"cache", "eviction", "lru", "hit", "miss", "ratio", "memory", "disk",
	"file", "buffer", "stream", "async", "sync", "parallel", "sequential",
	"cluster", "node", "partition", "consistency", "availability", "durability",
}

func randomDocument(rng *rand.Rand, wordCount int) string {
	words := make([]string, wordCount)
	for i := range words {
		words[i] = vocabulary[rng.Intn(len(vocabulary))]
	}
	// Occasionally add a sentence-level structure.
	return strings.Join(words, " ") + "."
}

func main() {
	addr := "localhost:50051"
	totalDocs := 10_000
	batchLog := 500

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect to %s: %v", addr, err)
	}
	defer conn.Close()

	client := pb.NewSearchServiceClient(conn)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	log.Printf("seeding %d documents into SearchCore at %s …", totalDocs, addr)
	start := time.Now()

	for i := 1; i <= totalDocs; i++ {
		wordCount := 20 + rng.Intn(181) // 20–200 words
		content := randomDocument(rng, wordCount)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := client.IndexDocument(ctx, &pb.IndexRequest{Content: content})
		cancel()

		if err != nil {
			log.Printf("doc %d: RPC error: %v", i, err)
			continue
		}
		if !resp.Success {
			log.Printf("doc %d: index failed: %s", i, resp.Message)
			continue
		}

		if i%batchLog == 0 {
			elapsed := time.Since(start)
			rps := float64(i) / elapsed.Seconds()
			fmt.Printf("  %5d / %d docs indexed  (%.0f docs/s)\n", i, totalDocs, rps)
		}
	}

	elapsed := time.Since(start)
	log.Printf("done: %d docs in %.1fs (%.0f docs/s)", totalDocs, elapsed.Seconds(), float64(totalDocs)/elapsed.Seconds())

	// Print final stats.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stats, err := client.Stats(ctx, &pb.StatsRequest{})
	if err != nil {
		log.Printf("stats RPC error: %v", err)
		return
	}
	fmt.Printf("\nCorpus stats:\n  total_docs:     %d\n  total_terms:    %d\n  avg_doc_length: %.1f\n",
		stats.TotalDocs, stats.TotalTerms, stats.AvgDocLength)
}
