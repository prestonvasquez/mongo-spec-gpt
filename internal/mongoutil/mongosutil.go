package mongoutil

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/vectorstores/mongovector"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	DefaultDatabaseName = "mongospecgpt"
	DefaultNamespace    = "specs"

	// We are going to use o4-mini for prototyping, which uses the
	// "text-embedding-3-small" embedding model which has dimensions of 1536. We
	// will use the dot product algorithm for similarity search.
	DefaultIndexName            = "vector_index_dotProduct_1536"
	DefaultIndexDimensions      = 1536
	DefaultEmbeddingPath        = "spec_embedding"
	DefaultOpenAIEmbeddingModel = "text-embedding-3-small"
	DefaultSimilarityAlgorithm  = "dotProduct"
	DefaultAPIVersion           = "2024-12-01-preview"
)

func searchIndexExists(ctx context.Context, coll *mongo.Collection, idx string) (bool, error) {
	view := coll.SearchIndexes()

	siOpts := options.SearchIndexes().SetName(idx).SetType("vectorSearch")
	cursor, err := view.List(ctx, siOpts)
	if err != nil {
		return false, fmt.Errorf("failed to list search indexes: %w", err)
	}

	if cursor == nil {
		return false, nil
	}

	for cursor.Next(ctx) {
		name := cursor.Current.Lookup("name").StringValue()
		queryable := cursor.Current.Lookup("queryable").Boolean()

		if name == idx && queryable {
			return true, nil
		}
	}

	return false, nil
}

// vectorField defines the fields of an index used for vector search.
type vectorField struct {
	Type          string `bson:"type,omitempty"`
	Path          string `bson:"path,omityempty"`
	NumDimensions int    `bson:"numDimensions,omitempty"`
	Similarity    string `bson:"similarity,omitempty"`
}

func createSearchIndex(ctx context.Context, client mongo.Client) error {
	logrus.Info("creating vector store collection")
	// Create the vectorstore collection
	err := client.Database(DefaultDatabaseName).CreateCollection(ctx, DefaultNamespace)
	if err != nil {
		return fmt.Errorf("failed to create vector store collection: %w", err)
	}

	coll := client.Database(DefaultDatabaseName).Collection(DefaultNamespace)

	if ok, _ := searchIndexExists(ctx, coll, DefaultIndexName); ok {
		logrus.Infof("search index %s already exists", DefaultIndexName)

		return nil
	}

	logrus.Infof("search index %s does not exist, creating it", DefaultIndexName)

	fields := []vectorField{}

	fields = append(fields, vectorField{
		Type:          "vector",
		Path:          DefaultEmbeddingPath,
		NumDimensions: DefaultIndexDimensions,
		Similarity:    DefaultSimilarityAlgorithm,
	}) // Append in case we want to add filters later.

	def := struct {
		Fields []vectorField `bson:"fields"`
	}{
		Fields: fields,
	}

	view := coll.SearchIndexes()

	siOpts := options.SearchIndexes().SetName(DefaultIndexName).SetType("vectorSearch")
	searchName, err := view.CreateOne(ctx, mongo.SearchIndexModel{Definition: def, Options: siOpts})
	if err != nil {
		return fmt.Errorf("failed to create the search index: %w", err)
	}

	// Await the creation of the index.
	var doc bson.Raw
	for doc == nil {
		cursor, err := view.List(ctx, options.SearchIndexes().SetName(searchName))
		if err != nil {
			return fmt.Errorf("failed to list search indexes: %w", err)
		}

		if !cursor.Next(ctx) {
			break
		}

		name := cursor.Current.Lookup("name").StringValue()
		queryable := cursor.Current.Lookup("queryable").Boolean()
		if name == searchName && queryable {
			doc = cursor.Current
		} else {
			time.Sleep(5 * time.Second)
		}
	}

	logrus.Infof("search index %s created", searchName)

	return nil
}

func Store(ctx context.Context, client *mongo.Client, embedder embeddings.Embedder) (*mongovector.Store, error) {
	if err := createSearchIndex(ctx, *client); err != nil {
		return nil, fmt.Errorf("failed to create search index: %w", err)
	}

	coll := client.Database(DefaultDatabaseName).Collection(DefaultNamespace)
	store := mongovector.New(coll, embedder,
		mongovector.WithIndex(DefaultIndexName),
		mongovector.WithPath(DefaultEmbeddingPath),
	)

	return &store, nil
}
