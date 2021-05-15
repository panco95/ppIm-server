package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"ppIm/app/http/api"
	"ppIm/app/http/api/v1/validate"
	"ppIm/app/model"
	"ppIm/lib"
	"ppIm/utils"
	"time"
)

type sign struct{}

var Sign sign

// 用户登录接口
func (sign) Login(ctx *gin.Context) {
	var Validate validate.Login
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")

	// 检测用户是否存在
	var user model.User
	var count int
	lib.Db.Where("username = ?", username).Select("id,username,password,password_salt,nickname,avatar,status").First(&user).Count(&count)
	if count < 1 {
		api.R(ctx, api.Fail, "用户不存在，请更换用户名后重试", nil)
		return
	}

	// 验证密码是否合法
	postPassword := utils.Md5(utils.Md5(password) + user.PasswordSalt)
	if postPassword != user.Password {
		api.R(ctx, api.Fail, "密码错误", nil)
		return
	}

	// 密码正确，更新登陆时间
	loginTime := time.Now().Format("2006-01-02 15:04:05")
	lib.Db.Model(&user).Updates(map[string]interface{}{"login_time": loginTime, "last_ip": ctx.ClientIP()})

	// 生成jwt token和用户信息给用户
	tokenString := api.MakeJwtToken(user.Id)
	api.R(ctx, api.Success, "登录成功", gin.H{
		"t": tokenString,
		"user": gin.H{
			"username": username,
			"nickname": user.Nickname,
			"avatar":   utils.QiNiuClient.FullPath(user.Avatar),
			"status":   user.Status,
		},
	})
}

//用户注册接口
func (sign) Register(ctx *gin.Context) {
	var Validate validate.Register
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")

	// 检测用户名是否存在
	var user model.User
	var count int
	lib.Db.Where("username = ?", username).First(&user).Count(&count)
	if count > 0 {
		api.R(ctx, api.Fail, "用户已存在，请更换用户名后重试", nil)
		return
	}

	// 新增用户数据，注册逻辑
	passwordSalt := utils.RandStr(6)
	password = utils.Md5(utils.Md5(password) + passwordSalt)

	user = model.User{
		Username:     username,
		Password:     password,
		Nickname:     "新用户" + username,
		PasswordSalt: passwordSalt,
		RegisterTime: time.Now().Format("2006-01-02 15:04:05"),
		LoginTime:    time.Now().Format("2006-01-02 15:04:05"),
		Avatar:       viper.GetString("qiniu.default_avatar"),
		LastIp:       ctx.ClientIP(),
	}
	if err := lib.Db.Create(&user).Error; err != nil {
		lib.Logger.Debugf(err.Error())
		api.R(ctx, api.Fail, "服务器错误", nil)
		return
	}

	// 生成jwt token和用户信息给用户
	tokenString := api.MakeJwtToken(user.Id)
	api.R(ctx, api.Success, "登录成功", gin.H{
		"t": tokenString,
		"user": gin.H{
			"username": username,
			"nickname": user.Nickname,
			"avatar":   utils.QiNiuClient.FullPath(user.Avatar),
			"status":   user.Status,
		},
	})
}
