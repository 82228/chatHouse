package main

import (
	"bufio"
	"chat/client/message"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// 登录或注册的网名
var Nickname string

func main() {
	dl := websocket.Dialer{}
	conn, _, _ := dl.Dial("ws://127.0.0.1:8888", nil)
	for {
		fmt.Println("[—————————欢迎来到聊天室系统—————————]")
		fmt.Println(" -----------请选择操作:------------")
		fmt.Println(" -----------1.注册新网名-----------")
		fmt.Println(" ----------2.登录已有网名----------")
		fmt.Println(" ------进入聊天室后输入3进行群聊-----")
		fmt.Println(" ------进入聊天室后输入4进行单聊-----")
		fmt.Println(" -进入聊天室后输入6查询用户活跃排行榜-")
		fmt.Println("——————————————————————————————————")
		reader := bufio.NewReader(os.Stdin)
		option, _ := reader.ReadString('\n')
		option = strings.TrimSpace(option) //去掉两端的空白字符，包括\r 和 \n
		fmt.Println(option)

		//将输入内容换成int类型的
		options, err := strconv.Atoi(option)
		if err != nil {
			log.Fatal(err)
		}

		switch options {
		case 1:
			register(conn)
		case 2:
			login(conn)
		default:
			fmt.Println("无效选项")

		}

		// 设置 Ping 处理器,
		conn.SetPingHandler(func(string) error {
			//log.Println("Received Ping from server")
			conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			err := conn.WriteMessage(websocket.PongMessage, nil)
			if err != nil {
				panic(err)
				return err
			}

			return nil
		})

		go read(conn)
		send(conn)

	}
}

// 注册
func register(conn *websocket.Conn) {
	for {
		var mes message.Message
		mes.Type = "注册"
		fmt.Println("请输入您想注册的新网名")
		Nickname, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		Nickname = Nickname[:len(Nickname)-1]
		mes.Content = Nickname

		com, err := json.Marshal(mes)
		if err != nil {
			log.Fatal(err)
		}
		//将commend发送至服务端
		conn.WriteMessage(websocket.TextMessage, []byte(com))

		_, p, _ := conn.ReadMessage()
		//log.Println(string(p))
		if string(p) == "该网名已存在，请重新输入:" {
			fmt.Println(string(p))
			continue
		} else if string(p) == "注册成功，您已进入聊天室" {
			fmt.Println("注册成功，您已进入聊天室")
			break
		}

	}

}

// 登录
func login(conn *websocket.Conn) {
	for {
		var mes message.Message
		mes.Type = "登录"
		fmt.Println("请输入您想登录的网名:")
		Nickname, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		Nickname = Nickname[:len(Nickname)-1]
		mes.Content = Nickname
		com, err := json.Marshal(mes)
		if err != nil {
			log.Fatal(err)
		}
		//将commend发送至服务端
		err = conn.WriteMessage(websocket.TextMessage, []byte(com))
		if err != nil {
			fmt.Println("发送失败")
			return
		}

		//读取服务端发送的信息
		_, p, _ := conn.ReadMessage()
		if string(p) == "登录失败,该网名不存在" {
			fmt.Println(string(p))
			continue
		} else if string(p) == "登录成功,您已进入聊天室" {
			fmt.Println(string(p))
			break
		} else if string(p) == "该用户已在线，请重新输入:" {
			fmt.Println(string(p))
			continue
		}
	}
}

// 发送消息
func send(conn *websocket.Conn) {
	for {
		var mes message.Message

		reader := bufio.NewReader(os.Stdin)
		l, _, _ := reader.ReadLine()
		option := string(l)
		switch option {
		case "3":
			mes.Type = "3"
			mes.Sender = Nickname
			fmt.Println("请输入你要群发的消息")
			fmt.Scanf("%s", &mes.Content)
		case "4":
			mes.Type = "4"
			mes.Sender = Nickname
			fmt.Println("请输入你要单聊的对象和消息内容:")
			fmt.Scanf("%s %s", &mes.Recipient, &mes.Content)
		case "6":
			mes.Type = "6"
		}
		p, _ := json.Marshal(mes)
		conn.WriteMessage(websocket.TextMessage, []byte(string(p)))
	}
}

// 读取消息
func read(conn *websocket.Conn) {
	var mes message.Message
	for {
		_, p, e := conn.ReadMessage()
		//fmt.Println(p)
		if e != nil {
			log.Println("客户端读取消息失败")
			conn.Close()
			break
		}
		err := json.Unmarshal(p, &mes)
		if err != nil {
			log.Println(err)
		}
		//fmt.Println(string(p))
		switch mes.Type {
		case "3":
			//广播消息
			fmt.Println(mes.Sender + ": " + mes.Content)
		case "4":
			//私发
			fmt.Println(mes.Sender + "向您单发了一条消息:" + mes.Content)
		case "5":
			//系统消息
			fmt.Println(mes.Content)
		case "6":
			fmt.Println(mes.Content)
		}
	}
}
