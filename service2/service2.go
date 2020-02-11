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

	// github.com/tarkvincactus/gotesttask/proto
	pb "GoTestTask/protobuf"

	"github.com/go-redis/redis/v7"
	"gopkg.in/yaml.v2"

	grpc "google.golang.org/grpc"
)

// в другом сревисе эта структуры называется configs - названия неконсистенты
type configFile struct {
	GrpcAddress string `yaml:"grpc_adress"`
	ListenPort  string `yaml:"listen_port"`
	RedisAddr   string `yaml:"redis_addr"`
	RedisPass   string `yaml:"redis_pass"`
	RedisDB     int    `yaml:"redis_db"`
}

// не точно, но вроде в таком просто случае можно 
// анмаршалить респонс без отдельной структуры вовсе
type JSONresponse struct {
	SearchTitle []string `json:"search_title"`
}

func getConfig() configFile {
	filename, _ := filepath.Abs("./config2.yaml")
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Println(err)
	}

	var conf configFile

	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		log.Println(err)
	}

	return conf
}

// ручкам лучше давать осмысленные названия, типа "Search"
func mainHandler(w http.ResponseWriter, r *http.Request, client pb.KeyWordMessagingClient, conf configFile) {
	var response JSONresponse

	key, ok := r.URL.Query()["keyword"]

	if !ok || len(key[0]) < 1 {
		log.Println("Url Param 'keyword' is missing")
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "Url Param 'keyword' is missing", http.StatusBadRequest)
		return
	}

	// вот то, что ты используешь контекст - очень круто, мы были удивлены
	//count timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	// используем функцию из другого сервиса. кажется, надо выносить в отдельный пакет
	// сервис2 по сути выполняет поиск, просто кешируя его после
	// выглядит не оч логично, что функция поиска называется setkeyword
	resp, err := client.SetKeyWord(ctx, &pb.KeyWordReq{Word: key[0]})
	if err != nil {
		log.Println("set keyword err: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//redis settings
	// вот это лучше перенести в меин
	rClient := redis.NewClient(&redis.Options{
		Addr:     conf.RedisAddr,
		Password: conf.RedisPass,
		DB:       conf.RedisDB,
	})

	// ху из рклиент - лучше уж оставить редисКлиент
	// 0, -1 
	val, err := rClient.LRange(resp.GetRedisKey(), 0, -1).Result()
	if err == redis.Nil {
		log.Println("redis key does not exist")
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if err != nil {
		log.Println("err - ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sort.Strings(val)
	response.SearchTitle = val

	byteResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(byteResponse)
	// return забыт
}

func main() {
	conf := getConfig()
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

	// log.Fatal(http.ListenAndServe(conf.ListenPort, nil))
	http.ListenAndServe(conf.ListenPort, nil)
}
