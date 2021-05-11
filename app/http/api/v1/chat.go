package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"path"
	"ppIm/app/http/api"
	"ppIm/app/http/api/v1/validate"
	"ppIm/app/model"
	"ppIm/app/service"
	"ppIm/app/websocket"
	"ppIm/lib"
	"ppIm/utils"
	"strconv"
	"strings"
	"time"
)

type chat struct{}

var Chat chat

// 发送好友消息
func (chat) SendToUser(ctx *gin.Context) {
	var Validate validate.SendToUser
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	toUid, _ := strconv.Atoi(ctx.PostForm("to_uid"))
	messageType, _ := strconv.Atoi(ctx.PostForm("type"))
	body := ctx.PostForm("body")

	uid := int(ctx.MustGet("id").(float64))

	var friendList model.FriendList
	var count int
	lib.Db.Where("uid = ? and f_uid = ?", uid, toUid).Select("id,uid,f_uid").First(&friendList).Count(&count)
	if count == 0 {
		api.R(ctx, api.Fail, "对方不是你的好友", nil)
		return
	} else {
		// 持久化消息记录
		chatMessage := model.ChatMessage{
			FromId:    uid,
			ToId:      toUid,
			Ope:       websocket.OpeFriend,
			Type:      messageType,
			Body:      body,
			Status:    0,
			CreatedAt: time.Now().Unix(),
		}
		lib.Db.Create(&chatMessage)
		// 发送实时消息
		message := websocket.Message{
			Cmd:    websocket.CmdReceiveFriendMessage,
			FromId: uid,
			ToId:   toUid,
			Ope:    websocket.OpeFriend,
			Type:   messageType,
			Body:   body,
		}
		websocket.SendToUser(toUid, message)

		api.Rt(ctx, api.Success, "发送成功", gin.H{"id": chatMessage.Id})
	}
}

// 撤回好友消息
func (chat) WithdrawFromUser(ctx *gin.Context) {
	var Validate validate.WithdrawFromUser
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	messageId, _ := strconv.Atoi(ctx.PostForm("message_id"))
	uid := int(ctx.MustGet("id").(float64))

	//查询消息记录
	var chatMessage model.ChatMessage
	lib.Db.Where("id = ? AND from_id = ? AND status <> ? AND ope = 0", messageId, uid, -1).First(&chatMessage)
	if chatMessage.Id == 0 {
		api.R(ctx, api.Fail, "消息不存在", nil)
		return
	}
	now := time.Now().Unix()
	// 判断消息发送是否超过2分钟
	if now > chatMessage.CreatedAt+120 {
		api.R(ctx, api.Fail, "消息超过2分钟无法撤回", nil)
		return
	}
	// 把消息状态改为已撤回
	lib.Db.Model(&chatMessage).Updates(map[string]interface{}{"status": -1})
	// 通知对方撤回消息
	message := websocket.Message{
		Cmd:    websocket.CmdWithdrawFriendMessage,
		FromId: uid,
		ToId:   chatMessage.ToId,
		Ope:    websocket.OpeFriend,
		Type:   websocket.TypePrompt,
		Body:   strconv.Itoa(chatMessage.Id),
	}
	websocket.SendToUser(chatMessage.ToId, message)

	api.Rt(ctx, api.Success, "撤回成功", gin.H{})
}

// 发送群消息
func (chat) SendToGroup(ctx *gin.Context) {
	var Validate validate.WithdrawFromUser
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	groupId, _ := strconv.Atoi(ctx.PostForm("group_id"))
	messageType, _ := strconv.Atoi(ctx.PostForm("type"))
	body := ctx.PostForm("body")

	uid := int(ctx.MustGet("id").(float64))

	var group model.Group
	lib.Db.Where("id = ?", groupId).First(&group)
	if group.Id == 0 {
		api.R(ctx, api.Fail, "群组不存在", nil)
		return
	}
	var groupUser model.GroupUser
	lib.Db.Where("group_id = ? AND user_id = ?", groupId, uid).First(&groupUser)
	if groupUser.Id == 0 {
		api.R(ctx, api.Fail, "您不是群成员", nil)
		return
	}

	// 持久化消息记录
	chatGroup := model.ChatMessage{
		FromId:    uid,
		ToId:      groupId,
		Ope:       websocket.OpeGroup,
		Type:      messageType,
		Body:      body,
		CreatedAt: time.Now().Unix(),
		Status:    0,
	}
	lib.Db.Create(&chatGroup)

	// 发送实时消息
	type result struct {
		UserId int
	}
	var groupUserList []result
	lib.Db.Raw("SELECT user_id FROM `group_user` WHERE group_id = >", groupId).Scan(&groupUserList)
	if len(groupUserList) > 0 {
		var userIdList []int
		for _, groupUser := range groupUserList {
			userIdList = append(userIdList, groupUser.UserId)
		}
		message := websocket.Message{
			Cmd:    websocket.CmdReceiveGroupMessage,
			FromId: uid,
			ToId:   groupId,
			Ope:    websocket.OpeGroup,
			Type:   messageType,
			Body:   body,
		}
		websocket.SendToGroup(userIdList, message)
	}

	api.Rt(ctx, api.Success, "发送成功", gin.H{"messageId": chatGroup.Id})
}

// 撤回群消息
func (chat) WithdrawFromGroup(ctx *gin.Context) {
	var Validate validate.WithdrawFromGroup
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	messageId, _ := strconv.Atoi(ctx.PostForm("message_id"))

	uid := int(ctx.MustGet("id").(float64))

	//查询消息记录
	var chatMessage model.ChatMessage
	lib.Db.Where("id = ? AND from_id = ? AND status <> ? AND ope = 1", messageId, uid, -1).First(&chatMessage)
	if chatMessage.Id == 0 {
		api.R(ctx, api.Fail, "消息不存在", nil)
		return
	}
	now := time.Now().Unix()
	// 判断消息发送是否超过2分钟
	if now > chatMessage.CreatedAt+120 {
		api.R(ctx, api.Fail, "消息超过2分钟无法撤回", nil)
		return
	}
	// 把消息状态改为已撤回
	lib.Db.Model(&chatMessage).Updates(map[string]interface{}{"status": -1})
	// 发送实时消息
	type result struct {
		UserId int
	}
	var groupUserList []result
	lib.Db.Raw("SELECT user_id FROM `group_user` WHERE group_id = >", chatMessage.ToId).Scan(&groupUserList)
	if len(groupUserList) > 0 {
		var userIdList []int
		for _, groupUser := range groupUserList {
			userIdList = append(userIdList, groupUser.UserId)
		}
		message := websocket.Message{
			Cmd:    websocket.CmdWithdrawGroupMessage,
			FromId: uid,
			ToId:   chatMessage.ToId,
			Ope:    websocket.OpeGroup,
			Type:   websocket.TypePrompt,
			Body:   strconv.Itoa(chatMessage.Id),
		}
		websocket.SendToGroup(userIdList, message)
	}

	api.Rt(ctx, api.Success, "撤回成功", gin.H{})
}

// 上传文件（聊天图片、语音、视频等）
func (chat) Upload(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		api.R(ctx, api.Fail, "请选择文件", nil)
		return
	}
	if file.Size/1024 > 20480 {
		api.R(ctx, api.Fail, "文件太大", nil)
		return
	}

	// 校验图片格式
	messageType, _ := strconv.Atoi(ctx.PostForm("type"))
	fileExt := strings.ToLower(path.Ext(file.Filename))
	switch messageType {
	case websocket.TypePicture:
		if fileExt != ".jpg" && fileExt != ".jpeg" && fileExt != ".bmp" && fileExt != ".png" && fileExt != ".gif" && fileExt != ".tif" && fileExt != ".webp" && fileExt != ".pcx" && fileExt != ".tga" && fileExt != ".exif" && fileExt != ".fpx" && fileExt != ".svg" && fileExt != ".wmf" {
			api.R(ctx, api.Fail, "图片格式不受支持", nil)
			return
		}
		break
	case websocket.TypeVoice:
		if fileExt != ".mp3" && fileExt != ".wma" && fileExt != ".aac" && fileExt != ".rm" && fileExt != ".ra" && fileExt != ".rmx" && fileExt != ".wav" && fileExt != ".aiff" && fileExt != ".ogg" && fileExt != ".amr" && fileExt != ".ape" && fileExt != ".flac" {
			api.R(ctx, api.Fail, "语音格式不受支持", nil)
			return
		}
		break
	case websocket.TypeVideo:
		if fileExt != ".avi" && fileExt != ".mp4" && fileExt != ".3gp" && fileExt != ".asf" && fileExt != ".wmv" && fileExt != ".rm" && fileExt != ".rmvb" && fileExt != ".flv" && fileExt != ".f4v" {
			api.R(ctx, api.Fail, "视频格式不受支持", nil)
			return
		}
		break
	case websocket.TypeFile:
		break
	default:
		api.R(ctx, api.Fail, "请选择文件", nil)
		return
	}

	id := int(ctx.MustGet("id").(float64))
	now := time.Now().Unix()
	localPath := fmt.Sprintf("runtime/upload/%d_%d%s", id, now, fileExt)
	if err := ctx.SaveUploadedFile(file, localPath); err != nil {
		api.R(ctx, api.Fail, "上传错误", nil)
		lib.Logger.Debugf(err.Error())
		return
	}
	uploadPath := fmt.Sprintf("chat/%d_%d%s", id, now, fileExt)
	err = service.UploadToQiNiu(uploadPath, localPath)
	if err != nil {
		api.R(ctx, api.Fail, "服务器错误", nil)
		return
	}

	api.Rt(ctx, api.Success, "上传成功", gin.H{
		"url": utils.QiNiuClient.FullPath(uploadPath),
	})
}
