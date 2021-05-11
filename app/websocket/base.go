package websocket

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"ppIm/app/service"
	"ppIm/lib"
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
		lib.Logger.Debugf(err.Error())
		return
	}

	c := Connection{
		Conn:    conn,
		Channel: make(chan []byte),
		Ctx:     ctx,
	}
	go ConnChannel(&c)

	var counter int
	ClientCounterLocker.Lock()
	ClientCounter++
	counter = ClientCounter
	Connections[counter] = &c
	ClientCounterLocker.Unlock()

	c.ClientId = GenClientId(counter)

	// 15秒内没收到token绑定成功的断开连接
	time.AfterFunc(15*time.Second, func() {
		if c.Uid == 0 {
			conn.Close()
		}
	})

	// 必须死循环，gin通过协程调用该handler函数，一旦退出函数，ws会被主动销毁
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			lib.Logger.Debugf(err.Error())
			break
		}
		c.Channel <- msg
	}

	ClientCounterLocker.Lock()
	delete(Connections, counter)
	if c.Uid > 0 {
		UidToClientIdLocker.Lock()
		delete(UidToClientId, c.Uid)
		UidToClientIdLocker.Unlock()
	}
	ClientCounterLocker.Unlock()
}

// 消息处理协程
func ConnChannel(c *Connection) {
	for {
		select {
		case msg := <-c.Channel:
			// json消息转换
			var message Message
			if err := json.Unmarshal(msg, &message); err != nil {
				// 接收到非json数据
				lib.Logger.Debugf(err.Error())
				message := Message{
					Cmd:  CmdFail,
					Body: "消息格式错误",
					Ope:  OpeSystem,
					Type: TypePrompt,
				}
				c.Conn.WriteJSON(message)
				continue
			}
			// bind绑定uid和client_id，这是必须绑定的才能通信的
			if message.Cmd == CmdSign {
				if c.Uid == 0 {
					jwtToken := message.Body
					id, err := service.ParseToken(c.Ctx, jwtToken)
					if err != "" {
						message := Message{
							Cmd:  CmdFail,
							Body: "认证失败",
							Ope:  OpeSystem,
							Type: TypePrompt,
						}
						c.Conn.WriteJSON(message)
					} else {
						c.Uid = id
						UidToClientIdLocker.Lock()
						UidToClientId[c.Uid] = c.ClientId // 认证成功后注册到已认证连接表，方便查询对应clientId
						UidToClientIdLocker.Unlock()
						message := Message{
							Cmd:  CmdSignSuccess,
							Body: "认证成功",
							Ope:  OpeSystem,
							Type: TypePrompt,
						}
						c.Conn.WriteJSON(message)
					}
				}
			} else {
				if c.Uid == 0 {
					message := Message{
						Cmd:  CmdFail,
						Body: "你还未认证",
						Ope:  OpeSystem,
						Type: TypePrompt,
					}
					c.Conn.WriteJSON(message)
				}
			}
		}
	}
}
