package validate

type UpdateNickname struct {
	Nickname string `form:"username" binding:"required,max=20,min=4"`
}

type RealNameVerify struct {
	RealName string `form:"real_name" binding:"required,max=20,min=4"`
	IDCard   string `form:"id_card" binding:"required,min=16"`
}

type UpdateLocation struct {
	Longitude string `form:"longitude" binding:"required,longitude"`
	Latitude  string `form:"latitude" binding:"required,latitude"`
}
