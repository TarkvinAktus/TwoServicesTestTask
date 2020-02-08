package main

import (
	pb "GoTestTask/protobuf"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path/filepath"

	"github.com/go-redis/redis/v7"
	"gopkg.in/yaml.v2"

	grpc "google.golang.org/grpc"
)

type googleResp struct {
	Items []item `json:"items"`
}

type item struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Link  string `json:"snippet"`
}

type configs struct {
	Port      string `yaml:"port"`
	RedisKey  string `yaml:"redis_key"`
	RedisAddr string `yaml:"redis_addr"`
	RedisPass string `yaml:"redis_pass"`
	RedisDB   int    `yaml:"redis_db"`
	URL       string `yaml:"url"`
	Cx        string `yaml:"cx"`
	Key       string `yaml:"key"`
}

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedKeyWordMessagingServer
}

func simpleSearchRequest(url string, q string, cx string, key string) string {
	q = "q=" + q
	cx = "cx=" + cx
	key = "key=" + key
	if q == "" || cx == "" || key == "" {
		log.Println("Error! missing URL params. Check config file 'URL', 'cx' or 'key' params")
	}
	return url + "?" + q + "&" + cx + "&" + key
}

func requestToGoogle(request string, conf configs) googleResp {

	var response googleResp

	resp, err := http.Get(simpleSearchRequest(conf.URL, request, conf.Cx, conf.Key))
	if err != nil {
		log.Println(err)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	if err := json.Unmarshal(bytes, &response); err != nil {
		log.Println(err)
	}

	return response
}

// SetKeyWord implements keyword.SetKeyWord
func (s *server) SetKeyWord(ctx context.Context, in *pb.KeyWordReq) (*pb.RedisKeyResp, error) {
	conf := getConfig()

	redisKey := conf.RedisKey

	//req to google api
	gResponse := requestToGoogle(in.GetWord(), conf)

	//redis
	client := redis.NewClient(&redis.Options{
		Addr:     conf.RedisAddr,
		Password: conf.RedisPass,
		DB:       conf.RedisDB,
	})

	client.Del(redisKey)

	for i := 0; i < 10; i++ {
		_ = client.LPush(redisKey, gResponse.Items[i].Title)
	}

	return &pb.RedisKeyResp{RedisKey: redisKey}, nil
}

func getConfig() configs {
	var conf configs

	filename, _ := filepath.Abs("./config1.yaml")
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
	}

	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		log.Println(err)
	}

	return conf
}

func main() {
	conf := getConfig()

	lis, err := net.Listen("tcp", conf.Port)
	if err != nil {
		log.Printf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterKeyWordMessagingServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
	}
}
