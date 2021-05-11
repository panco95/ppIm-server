package validate

type Login struct {
	Username string `form:"username" binding:"required,max=20,min=5"`
	Password string `form:"password" binding:"required,max=20,min=5"`
}

type Register struct {
	Username string `form:"username" binding:"required,max=20,min=5"`
	Password string `form:"password" binding:"required,max=20,min=5"`
}