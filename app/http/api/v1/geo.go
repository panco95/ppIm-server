package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"ppIm/app/http/api"
	"ppIm/app/http/api/v1/validate"
	"ppIm/app/model"
	"ppIm/lib"
	"ppIm/utils"
	"strconv"
)

type geo struct{}

var Geo geo

// 附近的人
func (geo) Users(ctx *gin.Context) {
	var Validate validate.Users
	if err := ctx.ShouldBind(&Validate); err != nil {
		api.R(ctx, api.Fail, "非法参数", gin.H{})
		return
	}
	longitude, _ := strconv.ParseFloat(ctx.PostForm("longitude"), 64)
	latitude, _ := strconv.ParseFloat(ctx.PostForm("latitude"), 64)

	// 距离范围，默认100
	distance := ctx.DefaultPostForm("distance","100")
	// 分页
	page := ctx.DefaultPostForm("page", "1")
	pageInt, _ := strconv.Atoi(page)
	size := 20
	from := (pageInt - 1) * size

	query := elastic.NewGeoDistanceQuery("location").Distance(distance + "km").Lat(latitude).Lon(longitude)
	sort := elastic.NewGeoDistanceSort("location").Point(latitude, longitude).Asc().DistanceType("arc").Unit("km")
	res, err3 := lib.Elasticsearch.Search().Index("user_location").Query(query).SortBy(sort).From(from).Size(size).Do(context.Background())
	if err3 != nil {
		api.R(ctx, api.Fail, "数据非法", nil)
		lib.Logger.Debugf(err3.Error())
		return
	}

	// 解析es数据数组
	type Data map[string]interface{}
	// es数组变量
	var data []Data

	// 循环es结果
	uid := int(ctx.MustGet("id").(float64))
	for _, hit := range res.Hits.Hits {
		var userLocation model.UserLocation
		err := json.Unmarshal(hit.Source, &userLocation) // json解析结果
		if err != nil {
			api.R(ctx, api.Fail, "服务器错误", nil)
			lib.Logger.Debugf(err.Error())
			return
		}
		var user model.User
		lib.Db.Where("id = ?", userLocation.Uid).First(&user)
		// 列表中排除自己
		if user.Id == uid {
			continue
		}
		temp := make(Data)

		// 距离换算公里/米
		distance, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", hit.Sort[0]), 64)
		var distanceEcho string
		if distance < 1 {
			distanceEcho = fmt.Sprintf("%dm", int(distance*1000))
		} else {
			distanceEcho = fmt.Sprintf("%.2fkm", distance)
		}

		temp = Data{
			"id":       user.Id,
			"username": user.Username,
			"nickname": user.Nickname,
			"avatar":   utils.QiNiuClient.FullPath(user.Avatar),
			"sex":      user.Sex,
			"distance": distanceEcho,
		}
		data = append(data, temp)
	}

	api.Rt(ctx, api.Success, "ok", gin.H{
		"list": data,
	})
}
