package main

import (
	//pb "GoTestTask/protobuf"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path/filepath"

	pb "github.com/TarkvinAktus/TwoServicesTestTask/protobuf"

	"github.com/go-redis/redis/v7"
	"gopkg.in/yaml.v2"

	grpc "google.golang.org/grpc"
)

type googleResp struct {
	Items []item `json:"items"`
}

type item struct {
	Title string `json:"title"`
}

type configs struct {
	ListenPort string `yaml:"listen_port"`
	RedisKey   string `yaml:"redis_key"`
	RedisAddr  string `yaml:"redis_addr"`
	RedisPass  string `yaml:"redis_pass"`
	RedisDB    int    `yaml:"redis_db"`
	ReqURL     string `yaml:"req_url"`
	ReqCx      string `yaml:"req_cx"`
	ReqKey     string `yaml:"req_key"`
	ReqNum     string `yaml:"req_num"`
}

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedKeyWordMessagingServer
}

func simpleSearchRequest(url string, q string, cx string, key string, num string) string {
	q = "q=" + q
	cx = "cx=" + cx
	key = "key=" + key
	num = "num=" + num
	if q == "" || cx == "" || key == "" {
		log.Println("Error! missing URL params. Check config file 'URL', 'cx', 'key' or 'num' params")
	}
	return url + "?" + q + "&" + cx + "&" + key + "&" + num
}

func requestToGoogle(request string, conf configs) (googleResp, error) {

	var response googleResp

	resp, err := http.Get(simpleSearchRequest(conf.ReqURL, request, conf.ReqCx, conf.ReqKey, conf.ReqNum))
	if err != nil {
		log.Println("http.Get", err)
		return response, err
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("ioutil.ReadAll", err)
		return response, err
	}

	if err := json.Unmarshal(bytes, &response); err != nil {
		log.Println("json.Unmarshal", err)
		return response, err
	}

	return response, nil
}

// SetKeyWord implements keyword.SetKeyWord
func (s *server) SetKeyWord(ctx context.Context, in *pb.KeyWordReq) (*pb.RedisKeyResp, error) {
	conf, err := getConfig()
	if err != nil {
		return &pb.RedisKeyResp{RedisKey: ""}, err
	}

	redisKey := conf.RedisKey

	//req to google api
	gResponse, err := requestToGoogle(in.GetWord(), conf)
	if err != nil {
		return &pb.RedisKeyResp{RedisKey: ""}, err
	}

	//redis
	client := redis.NewClient(&redis.Options{
		Addr:     conf.RedisAddr,
		Password: conf.RedisPass,
		DB:       conf.RedisDB,
	})

	client.Del(redisKey)
	//del err

	for i := range gResponse.Items {
		_ = client.LPush(redisKey, gResponse.Items[i].Title)
		//push err
	}

	return &pb.RedisKeyResp{RedisKey: redisKey}, err
}

func getConfig() (configs, error) {
	var conf configs

	filename, _ := filepath.Abs("./config1.yaml")
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

func main() {
	conf, err := getConfig()
	if err != nil {
		panic(err)
	}

	lis, err := net.Listen("tcp", conf.ListenPort)
	if err != nil {
		log.Printf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterKeyWordMessagingServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
	}
}
