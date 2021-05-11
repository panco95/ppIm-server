package validate

type Users struct {
	Longitude string `form:"longitude" binding:"required,longitude"`
	Latitude  string `form:"latitude" binding:"required,latitude"`
}