package validate

type Search struct {
	Word string `form:"word" binding:"required,max=10"`
}

type Add struct {
	Username string `form:"username" binding:"required,min=6,max=20"`
	Channel  string `form:"channel" binding:"required"`
	Reason   string `form:"reason" binding:"required"`
}

type AddHandle struct {
	FUid   string `form:"f_uid" binding:"required,numeric"`
	Status string `form:"status" binding:"required,numeric"`
}

type Del struct {
	FUid string `form:"f_uid" binding:"required,numeric"`
}
