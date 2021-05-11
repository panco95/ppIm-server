package validate

type Create struct {
	Name string `form:"name" binding:"required,max=20"`
}

type SearchG struct {
	Name string `form:"name" binding:"required,max=30"`
}

type Join struct {
	GroupId string `form:"group_id" binding:"required,numeric"`
}

type JoinHandle struct {
	JoinId string `form:"join_id" binding:"required,numeric"`
	Status string `form:"status" binding:"required,numeric"`
}

type Leave struct {
	GroupId string `form:"group_id" binding:"required,numeric"`
}

type Shot struct {
	GroupId string `form:"group_id" binding:"required,numeric"`
	UserId  string `form:"user_id" binding:"required,numeric"`
}
