package v1

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"path"
	"ppIm/app/http/api"
	"ppIm/app/http/api/v1/validate"
	"ppIm/app/model"
	"ppIm/app/service"
	"ppIm/lib"
	"ppIm/utils"
	"strconv"
	"strings"
	"time"
)

type user struct{}

var User user

// 用户更新昵称
func (user) UpdateNickname(ctx *gin.Context) {
	var Validate validate.UpdateNickname
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	nickname := ctx.PostForm("nickname")

	// jwt参数
	id := int(ctx.MustGet("id").(float64))
	// 更新用户昵称
	user := &model.User{Id: id}
	result := lib.Db.Model(&user).Update("nickname", nickname).RowsAffected
	api.Rt(ctx, api.Success, "设置成功", gin.H{"result": result})
}

// 用户更新头像
func (user) UpdateAvatar(ctx *gin.Context) {
	file, err := ctx.FormFile("avatar")
	if err != nil {
		api.R(ctx, api.Fail, "请选择图片", nil)
		return
	}
	if file.Size/1024 > 2048 {
		api.R(ctx, api.Fail, "文件太大", nil)
		return
	}

	// 保存头像文件，格式为id
	fileExt := strings.ToLower(path.Ext(file.Filename))
	if fileExt != ".jpg" && fileExt != ".jpeg" && fileExt != ".bmp" && fileExt != ".png" && fileExt != ".gif" && fileExt != ".tif" {
		api.R(ctx, api.Fail, "图片格式不受支持", nil)
		return
	}
	id := int(ctx.MustGet("id").(float64))
	now := time.Now().Unix()

	// 本地缓存地址
	localPath := fmt.Sprintf("runtime/upload/%d_%d%s", id, now, fileExt)
	if err := ctx.SaveUploadedFile(file, localPath); err != nil {
		api.R(ctx, api.Fail, "上传错误", nil)
		lib.Logger.Debugf(err.Error())
		return
	}
	// 七牛云上传地址
	uploadPath := fmt.Sprintf("avatar/%d_%d%s", id, now, fileExt)
	err = service.UploadToQiNiu(uploadPath, localPath)
	if err != nil {
		api.R(ctx, api.Fail, "服务器错误", nil)
		return
	}

	// 更新头像地址到数据库
	user := &model.User{Id: id}
	user.Avatar = uploadPath
	result := lib.Db.Model(&user).Update(user).RowsAffected

	api.Rt(ctx, api.Success, "设置成功", gin.H{"result": result})
}

// 实名认证
func (user) RealNameVerify(ctx *gin.Context) {
	var Validate validate.RealNameVerify
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	realName := ctx.PostForm("real_name")
	idCard := ctx.PostForm("id_card")

	//校验身份证信息
	x := []byte(idCard)
	if v := utils.IsValidCitizenNo(&x); !v {
		api.R(ctx, api.Fail, "身份证不合法", nil)
		return
	}

	// 获取身份证信息：性别、生日、省份
	_, _, sex, _ := utils.GetCitizenNoInfo(x)
	uSex := 0
	if sex == "男" {
		uSex = 1
	} else if sex == "女" {
		uSex = 2
	}

	// jwt参数
	id := int(ctx.MustGet("id").(float64))
	// 更新实名信息
	user := &model.User{Id: id}
	result := lib.Db.Model(&user).Updates(map[string]interface{}{"real_name": realName, "id_card": idCard, "sex": uSex}).RowsAffected
	api.Rt(ctx, api.Success, "实名认证成功", gin.H{"result": result})
}

//  更新最新地理位置及IP
func (user) UpdateLocation(ctx *gin.Context) {
	var Validate validate.UpdateLocation
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	longitude := ctx.PostForm("longitude")
	latitude := ctx.PostForm("latitude")

	id := int(ctx.MustGet("id").(float64))
	user := &model.User{Id: id}
	result := lib.Db.Model(&user).Updates(map[string]interface{}{"longitude": longitude, "latitude": latitude, "last_ip": ctx.ClientIP()}).RowsAffected
	api.Rt(ctx, api.Success, "更新位置成功", gin.H{"result": result})

	// 更新经纬度到es，用于后期查询
	data := fmt.Sprintf(`{
    "uid": "%d",
    "location": "%s,%s"
    }`, id, latitude, longitude)
	_, err := lib.Elasticsearch.Index().Index("user_location").Id(strconv.Itoa(int(id))).BodyJson(data).Do(context.Background())
	if err != nil {
		lib.Logger.Debugf(err.Error())
	}
}
