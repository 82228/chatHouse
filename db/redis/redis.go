package redis

import (
	"fmt"
	"github.com/go-redis/redis"
)

var Rdb *redis.Client

func InitRedis() {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err := Rdb.Ping().Result()
	if err != nil {
		fmt.Println("连接redis出错，错误为：", err)
		return
	}
	fmt.Println("连接成功")
}
