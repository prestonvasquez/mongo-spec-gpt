package mongospecgpt

import (
	"regexp"
	"strings"

	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

type Chunker interface {
	Chunk(values []string, metadata []map[string]any) ([]schema.Document, error)
}

type MarkdownChunker struct {
	splitter textsplitter.TextSplitter
}

var _ Chunker = &MarkdownChunker{}

func NewMarkdownChunker(modelName string, chunkSize, chunkOverlap int) *MarkdownChunker {
	s := textsplitter.NewMarkdownTextSplitter(
		textsplitter.WithModelName(modelName),
		textsplitter.WithChunkSize(chunkSize),
		textsplitter.WithChunkOverlap(chunkOverlap),
		textsplitter.WithHeadingHierarchy(true),
	)

	return &MarkdownChunker{splitter: s}
}

func (m *MarkdownChunker) Chunk(texts []string, metadata []map[string]any) ([]schema.Document, error) {
	return textsplitter.CreateDocuments(m.splitter, texts, metadata)
}

type SentenceChunker struct {
	MaxWords     int
	OverlapWords int
}

var _ Chunker = &SentenceChunker{}

func NewSentenceChunker(maxWords, overlapWords int) *SentenceChunker {
	return &SentenceChunker{MaxWords: maxWords, OverlapWords: overlapWords}
}

func (s *SentenceChunker) Chunk(texts []string, metadata []map[string]any) ([]schema.Document, error) {
	sentRegex := regexp.MustCompile(`(?m)([^\.!?]+[\.!?])`)

	var docs []schema.Document
	for i, text := range texts {
		sentences := sentRegex.FindAllString(text, -1) // Split into sentences.

		// Group sentences by word count.
		groups := groupByWords(sentences, s.MaxWords)
		for idx, grp := range groups {
			chunkText := strings.Join(grp, " ")
			meta := mergeMeta(metadata[i], map[string]any{"chunk_index": idx})
			docs = append(docs, schema.Document{
				PageContent: chunkText,
				Metadata:    meta,
			})
		}
	}

	return docs, nil
}

// groupByWords merges units into chunks each not exceeding maxWords.
func groupByWords(units []string, maxWords int) [][]string {
	var groups [][]string
	var current []string

	count := 0

	for _, unit := range units {
		words := len(strings.Fields(unit))
		if count+words > maxWords && len(current) > 0 {
			groups = append(groups, current)
			current = nil
			count = 0
		}
		current = append(current, strings.TrimSpace(unit))
		count += words
	}

	if len(current) > 0 {
		groups = append(groups, current)
	}

	return groups
}

// mergeMeta returns a new map combining base and extra keys.
func mergeMeta(base map[string]any, extra map[string]any) map[string]any {
	m := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		m[k] = v
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}
