package main

import (
	pb "GoTestTask/protobuf"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

type configFile struct {
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

func makeReqWithParams(params ...string) string {
	var result string
	//concat params

	return result
}

func requestToGoogle(request string, conf configFile) googleResp {

	var response googleResp

	//do concat func
	url := conf.URL

	q := "q=" + request
	cx := "cx=" + conf.Cx
	key := "key=" + conf.Key

	getReq := url + "?" + q + "&" + cx + "&" + key

	fmt.Println(getReq)

	resp, err := http.Get(getReq)
	if err != nil {
		fmt.Println(err)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	if err := json.Unmarshal(bytes, &response); err != nil {
		fmt.Println(err)
	}

	return response
}

// SetKeyWord implements keyword.SetKeyWord
func (s *server) SetKeyWord(ctx context.Context, in *pb.KeyWordReq) (*pb.RedisKeyResp, error) {
	conf := getConfig()

	redisKey := conf.RedisKey

	fmt.Println("Received: ", in.GetWord())

	//req to google api
	gResponse := requestToGoogle(in.GetWord(), conf)
	fmt.Println(gResponse)

	//redis
	client := redis.NewClient(&redis.Options{
		Addr:     conf.RedisAddr,
		Password: conf.RedisPass,
		DB:       conf.RedisDB,
	})

	err := client.Set(redisKey, "some_test_data", 0).Err()
	if err != nil {
		fmt.Println("redis SET err ", err)
	}

	return &pb.RedisKeyResp{RedisKey: redisKey}, nil
}

func getConfig() configFile {
	filename, _ := filepath.Abs("./config1.yaml")
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

func main() {
	conf := getConfig()

	lis, err := net.Listen("tcp", conf.Port)
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterKeyWordMessagingServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		fmt.Printf("failed to serve: %v", err)
	}
}
