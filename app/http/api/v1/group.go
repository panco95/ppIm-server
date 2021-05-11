package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"ppIm/app/http/api"
	"ppIm/app/http/api/v1/validate"
	"ppIm/app/model"
	"ppIm/app/websocket"
	"ppIm/lib"
	"strconv"
	"time"
)

type group struct{}

var Group group

// 创建群组
func (group) Create(ctx *gin.Context) {
	var Validate validate.Create
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	name := ctx.PostForm("name")

	uid := int(ctx.MustGet("id").(float64))

	var group model.Group
	var count int
	lib.Db.Where("name = ?", name).First(&group).Count(&count)
	if count > 0 {
		api.R(ctx, api.Fail, "群组已存在", nil)
		return
	}

	now := time.Now().Unix()
	trans := true
	// 事务
	lib.Db.Transaction(func(tx *gorm.DB) error {
		group.OUid = uid
		group.Name = name
		group.CreatedAt = now
		if err := tx.Create(&group).Error; err != nil {
			trans = false
			return err
		}

		var groupUser model.GroupUser
		groupUser.GroupId = group.Id
		groupUser.UserId = uid
		groupUser.JoinAt = now
		if err := tx.Create(&groupUser).Error; err != nil {
			trans = false
			return err
		}

		return nil
	})

	if trans == true {
		api.Rt(ctx, api.Success, "创建成功", gin.H{})
	} else {
		api.R(ctx, api.Fail, "创建失败", gin.H{})
	}
}

// 搜索群组
func (group) Search(ctx *gin.Context) {
	var Validate validate.SearchG
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	word := ctx.PostForm("word")

	var groups []model.Group
	lib.Db.Model(&model.Group{}).Where("name LIKE ?", "%"+word+"%").Find(&groups)
	api.R(ctx, api.Success, "查询成功", gin.H{"list": groups})
}

// 我的群组
func (group) My(ctx *gin.Context) {
	uid := int(ctx.MustGet("id").(float64))
	type Result struct {
		Id        int
		OUid      int
		Name      string
		CreatedAt int
		JoinAt    int
	}
	var result []Result
	lib.Db.Raw("SELECT g.id,g.o_uid,g.name,g.created_at,u.join_at FROM `group_user` AS u INNER JOIN `group` AS g ON u.group_id = g.id WHERE user_id = ?", uid).Scan(&result)
	api.R(ctx, api.Success, "获取成功", gin.H{"list": result})
}

// 申请加入群组
func (group) Join(ctx *gin.Context) {
	var Validate validate.Join
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	groupId, _ := strconv.Atoi(ctx.PostForm("group_id"))

	uid := int(ctx.MustGet("id").(float64))

	var group model.Group
	lib.Db.Where("id = ?", groupId).First(&group)
	if group.Id == 0 {
		api.R(ctx, api.Fail, "群组不存在", nil)
		return
	}
	if group.OUid == uid {
		api.R(ctx, api.Fail, "你已经是群主", nil)
		return
	}
	var groupUser model.GroupUser
	lib.Db.Where("group_id = ? and user_id = ?", groupId, uid).First(&groupUser)
	if groupUser.Id > 0 {
		api.R(ctx, api.Fail, "你已经在群组里", nil)
		return
	}
	var groupJoin model.GroupJoin
	lib.Db.Where("group_id = ? and user_id = ?", groupId, uid).First(&groupJoin)
	if groupJoin.Id > 0 {
		api.R(ctx, api.Fail, "你的申请加入群组请求已经在处理中", nil)
		return
	}
	groupJoin.GroupId = groupId
	groupJoin.UserId = uid
	groupJoin.JoinAt = time.Now().Unix()
	lib.Db.Create(&groupJoin)
	if groupJoin.Id > 0 {
		// 实时通知群组
		message := websocket.Message{
			Cmd:    websocket.CmdReceiveGroupJoin,
			FromId: uid,
			ToId:   group.Id,
			Ope:    websocket.OpeGroup,
			Type:   websocket.TypePrompt,
			Body:   "对方申请加入群组",
		}
		websocket.SendToUser(group.OUid, message)
		api.Rt(ctx, api.Success, "申请成功", gin.H{})
	} else {
		api.R(ctx, api.Fail, "申请失败", nil)
	}
}

// 申请加群列表
func (group) JoinList(ctx *gin.Context) {
	uid := int(ctx.MustGet("id").(float64))
	type Result struct {
		JoinId    int
		UserId    int
		GroupId   int
		Username  string
		Nickname  string
		Avatar    string
		GroupName string
		JoinAt    int
	}
	var result []Result
	lib.Db.Raw("SELECT j.id AS join_id,j.user_id,j.group_id,j.join_at,g.name AS group_name,u.username,u.nickname,u.avatar FROM `group` AS g INNER JOIN `group_join` AS j INNER JOIN `user` AS u ON g.id = j.group_id AND j.user_id = u.id WHERE g.o_uid = ? ORDER BY j.join_at", uid).Scan(&result)
	api.R(ctx, api.Success, "获取成功", gin.H{"list": result})
}

// 申请加群处理
func (group) JoinHandle(ctx *gin.Context) {
	var Validate validate.JoinHandle
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	joinId, _ := strconv.Atoi(ctx.PostForm("join_id"))
	status, _ := strconv.Atoi(ctx.PostForm("status")) // 1同意，-拒绝
	if status != 0 && status != 1 {
		api.R(ctx, api.Fail, "非法参数", nil)
		return
	}

	uid := int(ctx.MustGet("id").(float64))

	var groupJoin model.GroupJoin
	lib.Db.Where("id = ?", joinId).First(&groupJoin)
	if groupJoin.Id == 0 {
		api.R(ctx, api.Fail, "加群申请不存在", nil)
		return
	}
	var group model.Group
	lib.Db.Where("id = ?", groupJoin.GroupId).First(&group)
	if group.Id == 0 {
		api.R(ctx, api.Fail, "群组不存在", nil)
		return
	}
	if group.OUid != uid {
		api.R(ctx, api.Fail, "您不是群组，无法处理申请", nil)
		return
	}

	if status == 1 {
		// 同意加群
		trans := true
		// 事务
		lib.Db.Transaction(func(tx *gorm.DB) error {
			var groupUser model.GroupUser
			groupUser.GroupId = groupJoin.GroupId
			groupUser.UserId = groupJoin.UserId
			groupUser.JoinAt = time.Now().Unix()
			if err := tx.Create(&groupUser).Error; err != nil {
				trans = false
				return err
			}

			if err := tx.Delete(&groupJoin).Error; err != nil {
				trans = false
				return err
			}

			return nil
		})
		if trans == true {
			// 实时通知加入群组
			message := websocket.Message{
				Cmd:    websocket.CmdReceiveGroupJoinResult,
				FromId: groupJoin.GroupId,
				ToId:   groupJoin.UserId,
				Ope:    websocket.OpeGroup,
				Type:   websocket.TypePrompt,
				Body:   "群主同意您加入群组",
			}
			websocket.SendToUser(groupJoin.UserId, message)
			api.Rt(ctx, api.Success, "处理成功", gin.H{})
		} else {
			api.R(ctx, api.Fail, "处理失败", nil)
		}
	} else if status == 0 {
		// 拒绝加群
		lib.Db.Delete(&groupJoin)
		api.Rt(ctx, api.Success, "处理成功", gin.H{})
	}
}

// 退出群组
func (group) Leave(ctx *gin.Context) {
	var Validate validate.Leave
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	groupId, _ := strconv.Atoi(ctx.PostForm("group_id"))

	uid := int(ctx.MustGet("id").(float64))

	var groupUser model.GroupUser
	lib.Db.Where("uid = ? AND group_id = ?", uid, groupId).First(&groupUser)
	if groupUser.Id == 0 {
		api.R(ctx, api.Fail, "你不在群里", nil)
		return
	}
	lib.Db.Delete(&groupUser)
	api.Rt(ctx, api.Success, "退出群成功", gin.H{})
}

// 踢出群组
func (group) Shot(ctx *gin.Context) {
	var Validate validate.Shot
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	userId, _ := strconv.Atoi(ctx.PostForm("user_id"))
	groupId, _ := strconv.Atoi(ctx.PostForm("group_id"))

	uid := int(ctx.MustGet("id").(float64))

	var group model.Group
	lib.Db.Where("id = ?", groupId).First(&group)
	if group.Id == 0 {
		api.R(ctx, api.Fail, "群组不存在", nil)
		return
	}
	if group.OUid != uid {
		api.R(ctx, api.Fail, "你不是群主", nil)
		return
	}
	var groupUser model.GroupUser
	lib.Db.Where("user_id = ? AND group_id = ?", userId, groupId).First(&groupUser)
	if groupUser.Id == 0 {
		api.R(ctx, api.Fail, "用户不在群里", nil)
		return
	}
	lib.Db.Delete(&groupUser)
	// 实时通知用户被踢出群组
	message := websocket.Message{
		Cmd:    websocket.CmdReceiveGroupShot,
		FromId: groupId,
		ToId:   userId,
		Ope:    websocket.OpeGroup,
		Type:   websocket.TypePrompt,
		Body:   "你被踢出群组",
	}
	websocket.SendToUser(userId, message)
	api.Rt(ctx, api.Success, "踢出群成功", gin.H{})
}
