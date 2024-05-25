package pipeline

import (
	"errors"
	"go-gin-rest-api/models"

	"github.com/gin-gonic/gin"
	"github.com/linclin/fastflow/pkg/entity"
	"github.com/linclin/fastflow/pkg/mod"
	"gorm.io/gorm"
)

// @Summary [外部接口]创建流水线
// @Id CreatePipeline
// @Tags [外部接口]流水线
// @version 1.0
// @Accept application/x-json-stream
// @Param body body entity.Dag	true  "流水线id和参数"
// @Success 200 object models.Resp 返回列表
// @Failure 400 object models.Resp 查询失败
// @Security ApiKeyAuth
// @Router /api/v1/pipeline/upsert [post]
func UpsertPipeline(c *gin.Context) {
	var dag entity.Dag
	err := c.ShouldBindJSON(&dag)
	if err != nil {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
		return
	}
	oldDag, err := mod.GetStore().GetDag(dag.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := mod.GetStore().CreateDag(&dag); err != nil {
			models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
			return
		}
	} else {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
		return
	}
	if oldDag != nil {
		if err := mod.GetStore().UpdateDag(&dag); err != nil {
			models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
			return
		}
	}
	models.OkWithMessage("创建/更新成功", c)
}
