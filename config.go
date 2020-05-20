package gcron

import (
	"context"
	"github.com/go-ini/ini"
	"github.com/gomodule/redigo/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"time"
)

//配置是否已经初始化
var configIsLoad = false
var (
	RedisConfig = struct {
		Host        string
		Password    string
		MaxIdle     int
		MaxActive   int
		IdleTimeout time.Duration
		Db          int
		Timeout     int
		Zset        string
		Hash        string
	}{}
	MongoDBConfig = struct {
		Host             string
		Username         string
		Password         string
		RunLogCollection string
		ErrLogCollection string
		Database         string
	}{}
)

//初始化配置信息
func LoadConfig() {
	if configIsLoad {
		return
	}
	h, err := ini.Load("config.ini")
	if err != nil {
		log.Fatalln("配置文件读取错误 " + err.Error())
	}
	mapTo(h, "redis", &RedisConfig)
	mapTo(h, "mongodb", &MongoDBConfig)
	configIsLoad = true
}

//ini映射
func mapTo(h *ini.File, section string, v interface{}) {
	if err := h.Section(section).MapTo(v); err != nil {
		log.Fatal(err.Error())
	}
}

//================== 初始化redis连接======================
var redisIsLoad = false
var redisPool *redis.Pool

func connectRedisPool() {
	redisPool = &redis.Pool{
		Dial: func() (con redis.Conn, err error) {
			con, err = redis.Dial("tcp", RedisConfig.Host,
				redis.DialPassword(RedisConfig.Password),
				redis.DialDatabase(RedisConfig.Db),
				redis.DialConnectTimeout(time.Second*time.Duration(RedisConfig.Timeout)),
				redis.DialReadTimeout(time.Second*time.Duration(RedisConfig.Timeout)),
				redis.DialWriteTimeout(time.Second*time.Duration(RedisConfig.Timeout)))
			if err != nil {
				log.Fatalln("Redis连接错误", err.Error())
			}
			return con, err
		},
		MaxIdle:         RedisConfig.MaxIdle,
		MaxActive:       RedisConfig.MaxActive,
		IdleTimeout:     RedisConfig.IdleTimeout,
		Wait:            true,
		MaxConnLifetime: 0,
	}
	redisIsLoad = true
}

func RedisInstance() redis.Conn {
	if !redisIsLoad {
		connectRedisPool()
	}
	return redisPool.Get()
}

//======================初始化mongodb连接=====================

var mongoClient *mongo.Client
var mongoIsConnection = false

func connectMongodb() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	mongoClient, err := mongo.Connect(
		ctx,
		options.Client().
			ApplyURI("mongodb://localhost:27017").
			SetAuth(options.Credential{
				Username: "admin",
				Password: "123456",
			}))
	if err != nil {
		log.Fatalln("MongoDB连接失败 " + err.Error())
	}
	err = mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatalln("MongoDB无法Ping通 " + err.Error())
	}
	mongoIsConnection = true
}

//记录任务执行日志
func RunLog(statusCode int, log string) {
	if !mongoIsConnection {
		connectMongodb()
	}
	collection := mongoClient.Database(MongoDBConfig.Database).Collection(MongoDBConfig.RunLogCollection)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_, _ = collection.InsertOne(ctx, bson.M{"status_code": statusCode, "log": log, "at": time.Now().Format("2006-01-02 15:04:05")})
}

//记录程序错误日志
func ErrLog(log string) {
	if !mongoIsConnection {
		connectMongodb()
	}
	collection := mongoClient.Database(MongoDBConfig.Database).Collection(MongoDBConfig.ErrLogCollection)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_, _ = collection.InsertOne(ctx, bson.M{"log": log, "at": time.Now().Format("2006-01-02 15:04:05")})
}
