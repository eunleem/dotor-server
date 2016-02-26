package main

import (
	"fmt"
	"gopkg.in/redis.v3"
	"log"
)

var client *redis.Client

func newReditClient() *redis.Client {
	client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "xlaVkfVkfgo",
		DB:       0, // use default DB
	})

	if pong, err := client.Ping().Result(); err != nil {
		log.Println("Error connecting to Redis Server.")
		log.Print(err)
	} else {
		log.Println(pong, err)
	}

	return client
}

func resetRedis() {
	if client == nil {
		log.Println("Redis Client not initialized.")
		return
	}

	//client.Expire(key string, expiration time.Duration)
	client.FlushAll()
}

func exampleClient() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	err := client.Set("key", "value", 0).Err()
	if err != nil {
		panic(err)
	}

	val, err := client.Get("key").Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("key", val)

	val2, err := client.Get("key2").Result()
	if err == redis.Nil {
		fmt.Println("key2 does not exists")
	} else if err != nil {
		panic(err)
	} else {
		fmt.Println("key2", val2)
	}
	// Output: key value
	// key2 does not exists
}
