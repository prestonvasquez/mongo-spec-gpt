package mongospecgpt

import (
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

func (m *MarkdownChunker) Chunk(values []string, metadata []map[string]any) ([]schema.Document, error) {
	return textsplitter.CreateDocuments(m.splitter, values, metadata)
}
