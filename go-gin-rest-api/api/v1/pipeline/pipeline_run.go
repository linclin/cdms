package pipeline

import (
	"go-gin-rest-api/models"
	"go-gin-rest-api/models/pipeline"

	"github.com/gin-gonic/gin"
	"github.com/linclin/fastflow/pkg/entity"
	"github.com/linclin/fastflow/pkg/mod"
)

// @Summary [外部接口]启动流水线
// @Id RunPipeline
// @Tags [外部接口]流水线
// @version 1.0
// @Accept application/x-json-stream
// @Param body body pipeline.PipelineRun	true  "流水线id和参数"
// @Success 200 object models.Resp 返回列表
// @Failure 400 object models.Resp 查询失败
// @Security ApiKeyAuth
// @Router /api/v1/pipeline/run [post]
func RunPipeline(c *gin.Context) {
	var pipelineRun pipeline.PipelineRun
	err := c.ShouldBindJSON(&pipelineRun)
	if err != nil {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
		return
	}
	dag, err := mod.GetStore().GetDag(pipelineRun.Dag)
	if err != nil {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
		return
	}
	dagIns, err := dag.Run(entity.TriggerManually, pipelineRun.Var)
	if err != nil {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
		return
	}
	err = mod.GetStore().CreateDagIns(dagIns)
	if err != nil {
		models.FailWithDetailed(err, models.CustomError[models.NotOk], c)
		return
	}
	models.OkWithData(dagIns, c)
}
