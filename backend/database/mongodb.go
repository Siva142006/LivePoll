package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectMongo(ctx context.Context, uri, databaseName string) (*mongo.Database, error) {
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	if databaseName == "" {
		databaseName = "livepoll"
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetTimeout(10*time.Second))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	return client.Database(databaseName), nil
}
