package model

// 用户表
type User struct {
	Id           int
	Username     string
	Password     string
	PasswordSalt string
	Nickname     string
	Avatar       string
	Country      string
	City         string
	Sex          int
	RealName     string
	IdCard       string
	Status       int
	Longitude    string
	Latitude     string
	LastIp       string
	RegisterTime   string
	LoginTime      string
}
