package db

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var dbOnce sync.Once
var mongoClient *mongo.Client
var databaseMap sync.Map
var collMap sync.Map

// MongoClient returns database client instance
func MongoClient() *mongo.Client {
	if mongoClient != nil {
		return mongoClient
	}

	dbOnce.Do(func() {
		log.Printf("Establishing mongo connection ...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		clientOptions := options.Client()
		clientOptions.ApplyURI(os.Getenv("MONGO_URI"))
		clientOptions.SetAuth(options.Credential{
			Username: os.Getenv("MONGO_USER"),
			Password: os.Getenv("MONGO_PASS"),
		})

		client, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			log.Printf("Failed to establish mongo connection.")
			panic(err)
		}
		mongoClient = client
		log.Printf("Mongo connection established.")
	})

	return mongoClient
}

func CloseMongoConnection() {
	if mongoClient != nil {
		log.Printf("Closing mongo connection ...")
		err := mongoClient.Disconnect(context.Background())
		log.Printf("Mongo connection closed.")
		if err != nil {
			panic(err)
		}
	}
}

func DefaultCtx() context.Context {
	return context.TODO()
}

func DB() *mongo.Database {
	db, loaded := databaseMap.Load(os.Getenv("MONGO_DB"))
	if loaded {
		return db.(*mongo.Database)
	}
	mdb := MongoClient().Database(os.Getenv("MONGO_DB"))
	log.Printf("Mongo database instance created.")
	databaseMap.Store(os.Getenv("MONGO_DB"), mdb)
	return mdb
}

func Collection(name string) *mongo.Collection {
	coll, loaded := collMap.Load(name)
	if loaded {
		return coll.(*mongo.Collection)
	}
	mcoll := DB().Collection(name)
	log.Printf("Mongo database collection created.")
	collMap.Store(name, mcoll)
	return mcoll
}
