package main

import (
	"chat/db/redis"
	"chat/server/message"
	"encoding/json"
	"fmt"
	redis2 "github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// 一个读写缓冲区
var UP = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// zset的集合名
var zsetName = "sName"

// list的名字
var listName = "slist"

// map，用于存放用户名以及连接
var conns sync.Map

var mes message.Message

// 实现http升级为websocket
func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := UP.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	readSend(conn)

	//启动心跳检测
	go heartbeat(conn)

	// 设置 Pong 处理器，用于心跳检测
	conn.SetPongHandler(func(string) error {
		//log.Println("Received Pong from server")
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		return nil
	})
}

func readSend(conn *websocket.Conn) {
	//defer func() {
	//	if err := recover(); err != nil {
	//		log.Println(err)
	//	}
	//}()
	for {
		//fmt.Println("能读取到消息10---------------------------")
		_, comm, e := conn.ReadMessage()
		if e != nil {
			log.Println(e)
			conns.Delete(conn)
			return
		} else {
			fmt.Println("读取消息成功--------------------------------")
		}
		//将读取到的消息压入链表
		_, err := redis.Rdb.LPush(listName, comm).Result()
		if err != nil {
			log.Println("压入消息失败")
			log.Println(err)
		} else {
			fmt.Println("压入消息成功")
		}
		//取出消息
		v, err := redis.Rdb.BRPop(time.Second*5, listName).Result()
		if err != nil {
			log.Println("取出消息失败:", err)
			return
		} else {
			fmt.Println("取出消息成功")
			fmt.Println(v)
		}
		data := v[1]
		//反序列化
		err = json.Unmarshal([]byte(data), &mes)
		fmt.Println(mes)
		if err != nil {
			log.Println(err)
		}

		switch mes.Type {
		case "注册":
			_, err := redis.Rdb.ZRank(zsetName, mes.Content).Result()
			if err != nil {
				if err == redis2.Nil {
					//网名不存在，将网名存到redis中
					redis.Rdb.ZAdd(zsetName, redis2.Z{
						Score:  0,
						Member: mes.Content,
					}).Result()
					//注册成功，返回消息
					fmt.Println("注册成功，您已进入聊天室")
					conns.Store(mes.Content, conn) //进入聊天室
					conn.WriteMessage(websocket.TextMessage, []byte("注册成功，您已进入聊天室"))
				} else {
					log.Println("网名判断有误：", err)
				}
			} else {
				//网名已经存在，返回消息注册失败
				log.Printf("网名 %s 已存在\n", mes.Content)
				conn.WriteMessage(websocket.TextMessage, []byte("该网名已存在，请重新输入:"))
				//返回消息
			}
		case "登录":
			_, err := redis.Rdb.ZRank(zsetName, mes.Content).Result()
			if err != nil {
				if err == redis2.Nil {
					fmt.Printf("登录失败，网名 %s 不存在\n", mes.Content)
					//返回消息登录失败，没有存在该网名
					conn.WriteMessage(websocket.TextMessage, []byte("登录失败,该网名不存在"))
				} else {
					log.Println("网名判断有误：", err)
				}
			} else {
				//用户在线，就要重新输入
				if _, ok := conns.Load(mes.Content); ok {
					conn.WriteMessage(websocket.TextMessage, []byte("该用户已在线，请重新输入:"))
				} else {
					//用户不在线，而且已经注册过，返回消息登录成功
					conns.Store(mes.Content, conn) //将该客户端元素添加到切片当中
					conn.WriteMessage(websocket.TextMessage, []byte("登录成功,您已进入聊天室"))
				}
			}
		case "3":
			//广播消息,群聊
			redis.Rdb.ZIncrBy(zsetName, 1, mes.Sender).Result() //发送一次消息，分数+1
			//fmt.Println("-0-0-0-00000000000000000000" + mes.Sender)
			conns.Range(func(_, value interface{}) bool {
				con := value.(*websocket.Conn)
				con.WriteMessage(websocket.TextMessage, comm)
				return true
			})
			//panic("测试panic")
		case "4": //单聊
			var mess message.Message
			//如果对方在线
			if value, ok := conns.Load(mes.Recipient); ok { //找到接收方所在连接并发送
				fmt.Println(mes.Content + "--------------------------------")
				con := value.(*websocket.Conn)
				//将要发送的消息反序列化
				mes, err := json.Marshal(mes)
				if err != nil {
					log.Println(err)
				}
				_ = con.WriteMessage(websocket.TextMessage, []byte(mes))

			} else {
				mess.Type = "5"
				mess.Content = "发送失败"
				mess, err := json.Marshal(mess)
				if err != nil {
					log.Println(err)
				}
				conn.WriteMessage(websocket.TextMessage, mess)
			}
		case "6":
			zsetResult, e := redis.Rdb.ZRevRangeWithScores(zsetName, 0, -1).Result()
			if e != nil {
				return
			}

			//创建字符串，并向里面添加内容,实现用户排行榜
			var res strings.Builder
			res.WriteString("[------用户活跃排行榜------]\n")
			for i, z := range zsetResult {
				res.WriteString(fmt.Sprintf("第%d名: %s, 活跃度: %.0f\n", i+1, z.Member, z.Score))
				res.WriteString("-------------------------\n")
				//fmt.Println(z.Score, z.Member)
			}
			mes.Content = res.String()
			mes, err := json.Marshal(mes)
			if err != nil {
				log.Println(err)
			}
			conn.WriteMessage(websocket.TextMessage, []byte(mes))
		}
	}
}

// 心跳检测
func heartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(2 * time.Second) //时间间隔10秒钟。定时执行任务
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			conns.Range(func(key, value interface{}) bool {
				username := key.(string)
				conn := value.(*websocket.Conn)
				//发送心跳包
				err := conn.WriteMessage(websocket.PingMessage, []byte("asdasdas"))
				if err != nil {
					conns.Delete(username)
					//fmt.Println("wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww")
					conn.Close()
				}
				return true
			})

		}
	}
}

func main() {
	redis.InitRedis()

	http.HandleFunc("/", handler)
	http.ListenAndServe(":8888", nil)
}
