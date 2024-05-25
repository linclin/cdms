package pipeline

// 流水线
type PipelineRun struct {
	Dag string            `json:"dag" binding:"required" ` // 流水线名称
	Var map[string]string `json:"var" binding:"required" ` // 流水线参数
}
