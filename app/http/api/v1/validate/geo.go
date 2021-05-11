package validate

type Users struct {
	Longitude string `form:"Longitude" binding:"required,longitude"`
	Latitude  string `form:"Latitude" binding:"required,latitude"`
}