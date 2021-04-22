package router

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ppIm/api"
	v1 "ppIm/api/v1"
	"ppIm/middleware"
	"ppIm/servers"
	"ppIm/ws"
)

func SetRouter(r *gin.Engine) {
	// 全局跨域中间件
	r.Use(middleware.Cors)

	// 公开访问目录
	r.StaticFS("/public", http.Dir("./public"))

	// websocket连接
	r.GET("/ws", ws.WebsocketEntry)
	// websocket服务状态
	r.GET("/ws/status", ws.StatusApi)
	// 查询某用户是否在线
	r.GET("/ws/isOnline", ws.IsOnlineApi)

	// 集群服务器列表
	r.GET("/cluster/servers", servers.ApiQuery)

	// 首页
	r.GET("/", api.Welcome)
	// 未定义路由
	r.NoRoute(api.NotFound)
	r.NoMethod(api.NotFound)

	// 用户登录
	r.POST("/api/v1/login", v1.Sign.Login)
	// 用户注册
	r.POST("/api/v1/register", v1.Sign.Register)

	// 用户相关接口
	user := r.Group("/api/v1/user")
	user.Use(middleware.ValidateJwtToken)
	{
		// 设置昵称
		user.POST("/update/nickname", v1.User.UpdateNickname)
		// 设置头像
		user.POST("/update/avatar", v1.User.UpdateAvatar)
		// 实名认证
		user.POST("/update/realname", v1.User.RealNameVerify)
		// 更新用户位置（经纬度）
		user.POST("/update/location", v1.User.UpdateLocation)
	}

	// 位置相关接口
	geo := r.Group("/api/v1/geo")
	geo.Use(middleware.ValidateJwtToken)
	{
		// 附近的人
		geo.POST("/users", v1.Geo.Users)
	}

	// 聊天相关接口
	im := r.Group("/api/v1/im")
	im.Use(middleware.ValidateJwtToken)
	{

		// 好友系统
		friend := im.Group("/friend")
		// 搜索好友
		friend.POST("/search", v1.Friend.Search)
		// 好友列表
		friend.POST("/list", v1.Friend.List)
		// 添加好友
		friend.POST("/add/request", v1.Friend.Add)
		// 收到的添加请求列表
		friend.POST("/add/reqs", v1.Friend.AddReqs)
		// 处理好友请求
		friend.POST("/add/handle", v1.Friend.AddHandle)
		// 删除好友
		friend.POST("/del", v1.Friend.Del)

		// 群组系统
		group := im.Group("/group")
		// 群组搜索
		group.POST("/search", v1.Group.List)
		// 创建群组
		group.POST("/create", v1.Group.Create)
		// 我的群组
		group.POST("/my", v1.Group.My)
		// 请求加入群组
		group.POST("/join/request", v1.Group.Join)
		// 加入群组请求处理
		group.POST("/join/confirm", v1.Group.JoinHandle)
		// 离开群组
		group.POST("/leave", v1.Group.Leave)
		// 设置群成员
		group.POST("/set/member", v1.Group.SetMember)

		// 聊天系统
		chat := im.Group("/chat")
		// 发送消息给用户
		chat.POST("/send/user", v1.Chat.SendToUser)
		// 撤回消息
		chat.POST("/withdraw/user", v1.Chat.WithdrawFromUser)
	}

}
