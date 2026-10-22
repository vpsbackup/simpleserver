package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MClient *MongoClient

const (
	// Upper bound for a single Mongo call, so a slow backend cannot pin a handler
	// open forever (review #15).
	mongoOpTimeout      = 3 * time.Second
	mongoConnectTimeout = 5 * time.Second
)

type MongoClient struct {
	DB   string
	Coll string
	Addr string
	C    *mongo.Collection
}

func NewMongoClient(addr, db, coll string) *MongoClient {
	ctx, cancel := context.WithTimeout(context.Background(), mongoConnectTimeout)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(addr))
	if err != nil {
		log.Println("connect error:", addr, err)
		return nil
	}
	if err = client.Ping(ctx, nil); err != nil {
		log.Println("ping error:", addr, err)
		return nil
	}
	d := client.Database(db)
	c := d.Collection(coll)
	return &MongoClient{
		Addr: addr,
		DB:   db,
		Coll: coll,
		C:    c,
	}
}

func (mc *MongoClient) Insert(data interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := mc.C.InsertOne(ctx, data)
	if err != nil {
		log.Println("insert data error:", data, err)
		return err
	}
	return nil
}

func (mc *MongoClient) Find(filter, ret interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), mongoOpTimeout)
	defer cancel()
	c, err := mc.C.Find(ctx, filter)
	if err != nil {
		log.Println("find error:", err)
		return err
	}
	err = c.All(ctx, ret)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("document is nil")
			return nil
		}
		log.Println("find one error:", filter, err)
		return err
	}
	return nil
}

func (mc *MongoClient) FindSort(filter, sort, ret interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), mongoOpTimeout)
	defer cancel()
	opts := options.Find().SetSort(sort)
	c, err := mc.C.Find(ctx, filter, opts)
	if err != nil {
		log.Println("find error:", err)
		return err
	}
	err = c.All(ctx, ret)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("document is nil")
			return nil
		}
		log.Println("find one error:", filter, err)
		return err
	}
	return nil
}

func (mc *MongoClient) Del(filter interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), mongoOpTimeout)
	defer cancel()
	_, err := mc.C.DeleteOne(ctx, filter)
	if err != nil {
		log.Println("delete one error:", filter, err)
		return err
	}
	return nil
}
