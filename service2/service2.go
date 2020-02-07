package main

import (
	pb "GoTestTask/protobuf"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v7"
	"gopkg.in/yaml.v2"

	grpc "google.golang.org/grpc"
)

type configFile struct {
	Port        string `yaml:"port"`
	Address     string `yaml:"adress"`
	DefaultPort string `yaml:"default_port"`
	RedisAddr   string `yaml:"redis_addr"`
	RedisPass   string `yaml:"redis_pass"`
	RedisDB     int    `yaml:"redis_db"`
}

func getConfig() configFile {
	filename, _ := filepath.Abs("./config2.yaml")
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		fmt.Println(err)
	}

	var conf configFile

	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		fmt.Println(err)
	}

	return conf
}

func mainHandler(w http.ResponseWriter, r *http.Request, client pb.KeyWordMessagingClient, conf configFile) {
	// Contact the server and print out its response.

	//temporary
	arg := "wiki"

	//count timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	resp, err := client.SetKeyWord(ctx, &pb.KeyWordReq{Word: arg})
	if err != nil {
		fmt.Println("could not greet: ", err)
	}
	fmt.Println("Redis key: ", resp.GetRedisKey())

	//redis
	rClient := redis.NewClient(&redis.Options{
		Addr:     conf.RedisAddr,
		Password: conf.RedisPass,
		DB:       conf.RedisDB,
	})

	val, err := rClient.Get(resp.GetRedisKey()).Result()
	if err == redis.Nil {
		fmt.Println("redis key does not exist")
	} else if err != nil {
		panic(err)
	} else {
		fmt.Println("redis value - ", val)
	}

}

func main() {
	conf := getConfig()
	// Set up a connection to the server.
	conn, err := grpc.Dial(conf.Address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		fmt.Println("did not connect: ", err)
	}
	defer conn.Close()
	client := pb.NewKeyWordMessagingClient(conn)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mainHandler(w, r, client, conf)
	})

	http.ListenAndServe(conf.DefaultPort, nil)
}
