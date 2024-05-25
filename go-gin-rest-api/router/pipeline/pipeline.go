package pipeline

import (
	"go-gin-rest-api/api/v1/pipeline"
	"go-gin-rest-api/middleware"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

// 流水线执行接口
func InitPipelineRouter(r *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) (R gin.IRoutes) {
	router := r.Group("pipeline").Use(authMiddleware.MiddlewareFunc()).Use(middleware.CasbinMiddleware)
	{
		router.POST("/run", pipeline.RunPipeline)
		router.POST("/upsert", pipeline.UpsertPipeline)
	}
	return router
}
