package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MClient *MongoClient

type MongoClient struct {
	DB   string
	Coll string
	Addr string
	C    *mongo.Collection
}

func NewMongoClient(addr, db, coll string) *MongoClient {
	client, err := mongo.NewClient(options.Client().ApplyURI(addr))
	if err != nil {
		log.Println("new client error:", addr, err)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Println("connect error:", err)
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
	ctx := context.Background()
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

func (mc *MongoClient) Del(filter interface{}) error {
	ctx := context.Background()
	_, err := mc.C.DeleteOne(ctx, filter)
	if err != nil {
		log.Println("delete one error:", filter, err)
		return err
	}
	return nil
}
