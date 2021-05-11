package validate

type SendToUser struct {
	ToUid       string `form:"to_uid" binding:"required,numeric"`
	MessageType string `form:"type" binding:"required,numeric"`
	Body        string `form:"body" binding:"required,max=1024"`
}

type WithdrawFromUser struct {
	MessageId string `form:"message_id" binding:"required,numeric"`
}

type SendToGroup struct {
	GroupId     string `form:"group_id" binding:"required,numeric"`
	MessageType string `form:"type" binding:"required,numeric"`
	Body        string `form:"body" binding:"required,max=1024"`
}

type WithdrawFromGroup struct {
	MessageId string `form:"message_id" binding:"required,numeric"`
}
