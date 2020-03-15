package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"time"

	pb "github.com/TarkvinAktus/TwoServicesTestTask/protobuf"

	"github.com/go-redis/redis/v7"
	"gopkg.in/yaml.v2"

	grpc "google.golang.org/grpc"
)

type configFile struct {
	GrpcAddress string `yaml:"grpc_adress"`
	ListenPort  string `yaml:"listen_port"`
	RedisAddr   string `yaml:"redis_addr"`
	RedisPass   string `yaml:"redis_pass"`
	RedisDB     int    `yaml:"redis_db"`
}

type JSONresponse struct {
	SearchTitle []string `json:"search_title"`
}

func getConfig() (configFile, error) {
	var conf configFile

	filename, _ := filepath.Abs("./config2.yaml")
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return conf, err
	}

	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		log.Println(err)
		return conf, err
	}

	return conf, nil
}

func mainHandler(w http.ResponseWriter, r *http.Request, client pb.KeyWordMessagingClient, conf configFile) {
	var response JSONresponse

	key, ok := r.URL.Query()["keyword"]

	if !ok || len(key[0]) < 1 {
		log.Println("Url Param 'keyword' is missing")
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "Url Param 'keyword' is missing", http.StatusBadRequest)
		return
	}

	//count timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	resp, err := client.SetKeyWord(ctx, &pb.KeyWordReq{Word: key[0]})
	if err != nil {
		log.Println("service 1 err", err)
		http.Error(w, "service 1 err", http.StatusInternalServerError)
		return
	}

	//redis settings
	rClient := redis.NewClient(&redis.Options{
		Addr:     conf.RedisAddr,
		Password: conf.RedisPass,
		DB:       conf.RedisDB,
	})

	val, err := rClient.LRange(resp.GetRedisKey(), 0, -1).Result()
	if err == redis.Nil {
		log.Println("redis key does not exist")
		w.WriteHeader(http.StatusInternalServerError)
		return

	} else if err != nil {
		log.Println("err - ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		sort.Strings(val)
		response.SearchTitle = val

		byteResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(byteResponse)
	}

}

func main() {
	conf, err := getConfig()
	if err != nil {
		panic(err)
	}
	// Set up a connection to the server

	conn, err := grpc.Dial(conf.GrpcAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Println("did not connect: ", err)
	}
	defer conn.Close()

	client := pb.NewKeyWordMessagingClient(conn)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mainHandler(w, r, client, conf)
	})

	//log.Println("did not connect: ", err)

	http.ListenAndServe(conf.ListenPort, nil)
}
