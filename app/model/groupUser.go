package model

// 群组成员表
type GroupUser struct {
	Id      int
	GroupId int
	UserId  int
	JoinAt  int64
}
