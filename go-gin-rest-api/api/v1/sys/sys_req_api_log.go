package sys

import (
	"go-gin-rest-api/models"
	"go-gin-rest-api/models/sys"
	"go-gin-rest-api/pkg/global"

	"github.com/a8m/rql"
	"github.com/gin-gonic/gin"
)

// @Summary [系统内部]获取请求接口日志
// @Id GetReqApiLog
// @Tags [系统内部]日志
// @version 1.0
// @Accept application/x-json-stream
// @Param	body body  models.Req true "RQL查询json"
// @Success 200 object models.Resp 返回列表
// @Failure 400 object models.Resp 查询失败
// @Security ApiKeyAuth
// @Router /api/v1/reqapilog/list [post]
func GetReqApiLog(c *gin.Context) {
	rqlQueryParser, err := rql.NewParser(rql.Config{
		Model: sys.SysReqApiLog{},
	})
	if err != nil {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
		return
	}
	// 绑定参数
	var rqlQuery rql.Query
	err = c.ShouldBindJSON(&rqlQuery)
	if err != nil {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
		return
	}
	rqlParams, err := rqlQueryParser.ParseQuery(&rqlQuery)
	if err != nil {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
		return
	}

	list := make([]sys.SysReqApiLog, 0)
	query := global.Mysql
	query = query.Where(rqlParams.FilterExp, rqlParams.FilterArgs...)
	count := int64(0)
	err = query.Limit(rqlParams.Limit).Offset(rqlParams.Offset).Order(rqlParams.Sort).Find(&list).Count(&count).Error
	if err != nil {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
		return
	}
	// 返回分页数据
	var resp models.PageData
	// 设置分页参数
	resp.PageInfo.Offset = rqlParams.Offset
	resp.PageInfo.Limit = rqlParams.Limit
	resp.PageInfo.Total = count
	resp.PageInfo.SortBy = rqlParams.Sort
	// 设置数据列表
	resp.List = list
	models.OkWithData(resp, c)
}

// @Summary [系统内部]根据ID获取请求接口日志
// @Id GetReqApiLogById
// @Tags [系统内部]日志
// @version 1.0
// @Accept application/x-json-stream
// @Param	requestid path string true "RequestId"
// @Success 200 object models.Resp 返回列表
// @Failure 400 object models.Resp 查询失败
// @Security ApiKeyAuth
// @Router /api/v1/reqapilog/get/{requestid} [get]
func GetReqApiLogById(c *gin.Context) {
	list := make([]sys.SysReqApiLog, 0)
	requestid := c.Param("requestid")
	err := global.Mysql.Where("RequestId = ?", requestid).Find(&list).Error
	if err != nil {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
	} else {
		models.OkWithData(list, c)
	}
}
