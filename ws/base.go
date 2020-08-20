package ws

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"ppIm/middleware"
	"time"
)

// 升级http为websocket服务
var WebsocketUpgrade = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 5 * time.Second,
	// 取消ws跨域校验
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// websocket服务入口
func WebsocketEntry(ctx *gin.Context) {
	w := ctx.Writer
	r := ctx.Request
	var conn *websocket.Conn
	var err error
	conn, err = WebsocketUpgrade.Upgrade(w, r, nil) // 升级为websocket协议
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	// 连接数+1
	OnlinesMutex.Lock()
	Onlines++
	OnlinesMutex.Unlock()

	// 是否认证绑定
	isBind := false
	// 15秒内没收到token绑定成功的断开连接
	time.AfterFunc(15*time.Second, func() {
		if !isBind {
			conn.Close()
		}
	})

	// 必须死循环，gin通过协程调用该handler函数，一旦退出函数，ws会被主动销毁
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// json消息转换
		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			// 接收到非json数据
			fmt.Println("[Websocket]Message json.Unmarshal fail: " + string(msg))
			conn.WriteJSON(WsMsg("error", 500, "Message is not json,failed!", nil))
			continue
		}

		// bind绑定uid和client_id，这是必须绑定的才能通信的
		if message.Route == "bind" {
			if !isBind {
				jwtToken := message.Data["token"]
				id, err := middleware.ParseToken(ctx, jwtToken.(string))
				if err != nil {
					fmt.Println(err)
				}
				var c Connection
				c.ClientId = GenClientId()
				c.Uid = id
				c.Conn = conn
				Connections[c.Uid] = c
				isBind = true
				conn.WriteJSON(WsMsg("bind", 200, "success", nil))
			}
		} else {
			if isBind {
				Receive(conn, message)
			} else {
				conn.WriteJSON(WsMsg("error", 500, "You are not bind!", nil))
			}
		}
	}
	// 退出循环，连接关闭，连接数-1
	OnlinesMutex.Lock()
	Onlines--
	OnlinesMutex.Unlock()
}
