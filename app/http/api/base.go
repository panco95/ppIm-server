package api

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"ppIm/lib"
	"time"
)

func Welcome(ctx *gin.Context) {
	R(ctx, Success, "欢迎使用，承接项目开发，请联系微信：freePan_1995", nil)
}

func NotFound(ctx *gin.Context) {
	R(ctx, Fail, "Not Found", nil)
}

// 响应封装（不带新token）
func R(ctx *gin.Context, code int, msg string, data gin.H) {
	data = gin.H{"code": code, "msg": msg, "data": data}
	ctx.JSON(http.StatusOK, data)
}

// 响应封装（带新token）
func Rt(ctx *gin.Context, code int, msg string, data gin.H) {
	// 实现7天内活跃用户无需重新登陆：请求成功响应后生成新token
	id := int(ctx.MustGet("id").(float64))
	data["t"] = MakeJwtToken(id)

	data = gin.H{"code": code, "msg": msg, "data": data}
	ctx.JSON(http.StatusOK, data)
}

// 生成用户token
func MakeJwtToken(id int) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": id,
		"at": time.Now().Unix(), // token有效期：一天
	})
	tokenString, err := token.SignedString(lib.JwtHmacSampleSecret)
	if err != nil {
		lib.Logger.Debugf(err.Error())
	}

	return tokenString
}
