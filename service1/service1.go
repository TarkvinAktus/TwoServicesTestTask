package main

import (
	// github.com/tarkvincactus/gotesttask/proto
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

// забыл удалить комментарии из какой то копипасты
// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedKeyWordMessagingServer
}

// почему он симпл? название не несет особого смысла
// мб лучше makeGoogleRequestParams
func simpleSearchRequest(url string, q string, cx string, key string, num string) string {
	q = "q=" + q
	cx = "cx=" + cx
	key = "key=" + key
	num = "num=" + num
	if q == "" || cx == "" || key == "" {
		// если тут произошла ошибка - мы вряд ли хотим продолжать дальше
		// так что нужно обрывать работу функции и возвращать ошибку errors.New()
		log.Println("Error! missing URL params. Check config file 'URL', 'cx', 'key' or 'num' params")
	}
	return url + "?" + q + "&" + cx + "&" + key + "&" + num
}

// реквестТуГугл я английский говорить
// search()
func requestToGoogle(request string, conf configs) googleResp {

	var response googleResp

	resp, err := http.Get(simpleSearchRequest(conf.ReqURL, request, conf.ReqCx, conf.ReqKey, conf.ReqNum))
	if err != nil {
		// не смогли получить данные - возвращаем ошибку
		// зачем нам анмаршаллить, если данных нет?
		log.Println(err)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// не смогли распарсить - возвращаем ошибку
		log.Println(err)
	}

	if err := json.Unmarshal(bytes, &response); err != nil {
		// не смогли распарсить данные - надо возвращать ошибку
		// тем самым останавливая дальнейшую работу функции
		log.Println(err)
	}

	return response
}

// SetKeyWord implements keyword.SetKeyWord
// почему она экспортируемая? 
// почему в функции, которая в названии подразумевает запись кейворда в редис 
// делаем еще и хттп запрос? какая здесь логика?
func (s *server) SetKeyWord(ctx context.Context, in *pb.KeyWordReq) (*pb.RedisKeyResp, error) {
	conf := getConfig()

	// эта переменная не очень нужна
	redisKey := conf.RedisKey

	//req to google api
	// googleResp
	gResponse := requestToGoogle(in.GetWord(), conf)

	//redis
	//название redisClient больше подойдет
	client := redis.NewClient(&redis.Options{
		Addr:     conf.RedisAddr,
		Password: conf.RedisPass,
		DB:       conf.RedisDB,
	})

	client.Del(redisKey)

	for i := range gResponse.Items {
		_ = client.LPush(redisKey, gResponse.Items[i].Title)
	}

	// в возращаемых аргументах предусмотрена ошибка, хотя в теле метода обработки ошибок нет
	return &pb.RedisKeyResp{RedisKey: redisKey}, nil
}

func getConfig() configs {
	var conf configs

	filename, _ := filepath.Abs("./config1.yaml")
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		// явно не сможем продолжить работать без конфига
		log.Println(err)
	}

	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		//тут явно мы работать дальше не сможем - надо падать
		log.Println(err)
	}

	return conf
}

func main() {
	conf := getConfig()

	// lis... что побудило тебя сократить именно так?(
	lis, err := net.Listen("tcp", conf.ListenPort)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		// log.Fatal!
	}
	s := grpc.NewServer()
	pb.RegisterKeyWordMessagingServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
		// log.Fatal!!!
	}
}
