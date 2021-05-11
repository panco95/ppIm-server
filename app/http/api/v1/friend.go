package v1

import (
	"github.com/gin-gonic/gin"
	"ppIm/app/http/api"
	"ppIm/app/http/api/v1/validate"
	"ppIm/app/model"
	"ppIm/app/websocket"
	"ppIm/lib"
	"strconv"
	"time"
)

type friend struct{}

var Friend friend

// 搜索好友
func (friend) Search(ctx *gin.Context) {
	var Validate validate.Search
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	word := ctx.PostForm("word")

	uid := int(ctx.MustGet("id").(float64))

	type APIUser struct {
		Username string
		Nickname string
		Avatar   string
		Sex      int
	}
	var users []APIUser
	lib.Db.Model(&model.User{}).Where("(username LIKE ? or nickname LIKE ?) and id <> ?", "%"+word+"%", "%"+word+"%", uid).Scan(&users)
	api.Rt(ctx, api.Success, "请求成功", gin.H{"list": users})
}

// 好友列表
func (friend) List(ctx *gin.Context) {
	uid := int(ctx.MustGet("id").(float64))
	type Result struct {
		Id       int
		Nickname string
		Username string
		Avatar   string
		Sex      int
	}
	var result Result
	var results []Result
	rows, err := lib.Db.Raw("select u.id,u.nickname,u.username,u.avatar,u.sex from friend_list as f join user as u on f.f_uid = u.id where f.uid = ?", uid).Rows()
	if err != nil {
		lib.Logger.Debugf(err.Error())
		api.R(ctx, api.Fail, "服务器错误", nil)
		return
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&result.Id, &result.Nickname, &result.Username, &result.Avatar, &result.Sex)
		results = append(results, result)
	}
	api.Rt(ctx, api.Success, "请求成功", gin.H{"lists": results})
}

// 添加好友
func (friend) Add(ctx *gin.Context) {
	var Validate validate.Add
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	username := ctx.PostForm("username")
	channel := ctx.PostForm("channel")
	reason := ctx.PostForm("reason")

	uid := int(ctx.MustGet("id").(float64))

	var user model.User
	var count int
	lib.Db.Where("username = ?", username).Select("id,username").First(&user).Count(&count)
	if count < 1 {
		api.R(ctx, api.Fail, "用户未找到", nil)
		return
	}
	if user.Id == uid {
		api.R(ctx, api.Fail, "不能添加自己", nil)
		return
	}

	var friendList model.FriendList
	lib.Db.Where("uid = ? and f_uid = ?", uid, user.Id).Select("id,uid,f_uid").First(&friendList).Count(&count)
	if count > 0 {
		api.R(ctx, api.Fail, "对方已经是好友", nil)
		return
	}

	var friendAdd model.FriendAdd
	lib.Db.Where("uid = ? and f_uid = ? and status = 0", uid, user.Id).Select("id,uid,f_uid").First(&friendAdd).Count(&count)
	if count > 0 {
		api.R(ctx, api.Fail, "请等待好友同意", nil)
		return
	}

	friendAdd.Uid = uid
	friendAdd.FUid = user.Id
	friendAdd.Reason = reason
	friendAdd.Channel = channel
	friendAdd.RequestAt = time.Now().Unix()
	if err := lib.Db.Create(&friendAdd).Error; err != nil {
		api.R(ctx, api.Fail, "服务器错误", nil)
		lib.Logger.Debugf(err.Error())
		return
	}

	// 实时通知用户添加请求
	var me model.User
	lib.Db.Where("id = ?", uid).First(&me)
	message := websocket.Message{
		Cmd:    websocket.CmdReceiveFriendAdd,
		FromId: uid,
		ToId:   user.Id,
		Ope:    websocket.OpeFriend,
		Type:   websocket.TypePrompt,
		Body:   "您收到一条好友添加请求",
	}
	websocket.SendToUser(user.Id, message)

	api.Rt(ctx, api.Success, "成功发送添加请求", gin.H{})
}

// 收到的好友请求列表
func (friend) AddReqs(ctx *gin.Context) {
	uid := int(ctx.MustGet("id").(float64))
	type Result struct {
		Id        int
		Nickname  string
		Username  string
		Avatar    string
		Channel   string
		Reason    string
		RequestAt string
	}
	var result Result
	var results []Result
	rows, err := lib.Db.Raw("select u.id,u.nickname,u.username,u.avatar,f.channel,f.reason,f.request_at from friend_add as f join user as u on f.uid = u.id  where f.f_uid = ? order by request_at desc", uid).Rows()
	if err != nil {
		lib.Logger.Debugf(err.Error())
		api.R(ctx, api.Fail, "服务器错误", nil)
		return
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&result.Id, &result.Nickname, &result.Username, &result.Avatar, &result.Channel, &result.Reason, &result.RequestAt)
		results = append(results, result)
	}
	api.Rt(ctx, api.Success, "请求成功", gin.H{"list": results})
}

// 处理收到的好友请求
func (friend) AddHandle(ctx *gin.Context) {
	var Validate validate.AddHandle
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	fUid, _ := strconv.Atoi(ctx.PostForm("f_uid"))
	status, _ := strconv.Atoi(ctx.PostForm("status"))
	if status != 1 && status != -1 {
		api.R(ctx, api.Fail, "非法参数", nil)
		return
	}

	uid := int(ctx.MustGet("id").(float64))
	var friendAdd model.FriendAdd
	lib.Db.Where("uid = ? and f_uid = ? and status = ?", fUid, uid, 0).First(&friendAdd)
	if friendAdd.Id == 0 {
		api.R(ctx, api.Fail, "添加好友请求不存在", nil)
		return
	}
	now := time.Now().Unix()
	lib.Db.Model(&friendAdd).Updates(map[string]interface{}{"status": status, "pass_at": now})
	if status == -1 {
		api.Rt(ctx, api.Success, "处理成功", gin.H{})
		return
	}

	// 通过验证后进行好友数据写入
	friendList1 := &model.FriendList{
		Uid:       uid,
		FUid:      fUid,
		Channel:   friendAdd.Channel,
		Reason:    friendAdd.Reason,
		Role:      1,
		CreatedAt: now,
	}
	friendList2 := &model.FriendList{
		Uid:       fUid,
		FUid:      uid,
		Channel:   friendAdd.Channel,
		Reason:    friendAdd.Reason,
		Role:      2,
		CreatedAt: now,
	}
	err1 := lib.Db.Create(&friendList1).Error
	err2 := lib.Db.Create(&friendList2).Error
	if err1 != nil || err2 != nil {
		api.R(ctx, api.Fail, "未知错误", nil)
		return
	}

	// 实时通知对方通过了好友请求
	var me model.User
	lib.Db.Where("id = ?", uid).First(&me)
	message := websocket.Message{
		Cmd:    websocket.CmdReceiveFriendAddResult,
		FromId: uid,
		ToId:   fUid,
		Ope:    websocket.OpeFriend,
		Type:   websocket.TypePrompt,
		Body:   "对方通过了你的好友请求",
	}
	websocket.SendToUser(fUid, message)

	api.Rt(ctx, api.Success, "处理成功", gin.H{})
}

// 删除好友
func (friend) Del(ctx *gin.Context) {
	var Validate validate.Del
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	fUid, _ := strconv.Atoi(ctx.PostForm("f_uid"))

	uid := int(ctx.MustGet("id").(float64))

	var friendList model.FriendList
	lib.Db.Where("uid = ? and f_uid = ?", uid, fUid).First(&friendList)
	if friendList.Id == 0 {
		api.R(ctx, api.Fail, "对方不是你的好友", nil)
		return
	}
	err := lib.Db.Delete(&friendList).Error
	if err != nil {
		lib.Logger.Debugf(err.Error())
		api.R(ctx, api.Fail, "删除失败", nil)
		return
	}
	api.Rt(ctx, api.Success, "删除成功", gin.H{})
}
