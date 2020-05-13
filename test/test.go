package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

func main() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(
		ctx,
		options.Client().
			ApplyURI("mongodb://localhost:27017").
			SetAuth(options.Credential{
				Username: "admin",
				Password: "123456",
			}))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	collection := client.Database("cron_log").Collection("test1")
	_, err = collection.InsertOne(ctx, bson.M{"name": "pi", "value": 3.14159})
	if err != nil {
		fmt.Println(err.Error())
	}
}
