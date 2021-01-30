package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"github.com/spf13/viper"
	"ppIm/api"
	"ppIm/global"
	"ppIm/model"
	"strconv"
)

// 附近的人
func GeoUsers(ctx *gin.Context) {
	// 经纬度校验
	longitude, err1 := strconv.ParseFloat(ctx.PostForm("longitude"), 64)
	latitude, err2 := strconv.ParseFloat(ctx.PostForm("latitude"), 64)
	if err1 != nil || err2 != nil {
		api.R(ctx, global.FAIL, "数据非法", nil)
		return
	}
	// 距离范围，默认100
	distance := ctx.PostForm("distance")
	if distance == "" {
		distance = "100"
	}
	// 分页
	page := ctx.PostForm("page")
	if page == "" {
		page = "1"
	}
	pageInt, _ := strconv.Atoi(page)
	size := 20
	from := (pageInt - 1) * size

	query := elastic.NewGeoDistanceQuery("location").Distance(distance + "km").Lat(latitude).Lon(longitude)
	sort := elastic.NewGeoDistanceSort("location").Point(latitude, longitude).Asc().DistanceType("arc").Unit("km")
	res, err3 := global.Elasticsearch.Search().Index("user_location").Query(query).SortBy(sort).From(from).Size(size).Do(context.Background())
	if err3 != nil {
		api.R(ctx, global.FAIL, "数据非法", nil)
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
			fmt.Println(err)
		}
		var user model.User
		global.Mysql.Where("id = ?", userLocation.Uid).First(&user)
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
			"avatar":   viper.GetString("app.domain") + user.Avatar,
			"sex":      user.Sex,
			"distance": distanceEcho,
		}
		data = append(data, temp)
	}

	api.Rt(ctx, global.SUCCESS, "ok", gin.H{
		"list": data,
	})
}
