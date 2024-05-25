package mongo

import (
	"fmt"
	"strings"
	"time"

	"github.com/linclin/fastflow/pkg/entity"
	"github.com/linclin/fastflow/pkg/event"
	"github.com/linclin/fastflow/pkg/mod"
	"github.com/shiningrush/goevent"
	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// StoreOption
type StoreOption struct {
	// mysql connection string
	ConnStr string
	// the prefix will append to the database
	Prefix string
}

// Store
type Store struct {
	opt            *StoreOption
	dagClsName     string
	dagInsClsName  string
	taskInsClsName string
	db             *gorm.DB
}

// NewStore
func NewStore(option *StoreOption) *Store {
	return &Store{
		opt: option,
	}
}

// Init store
func (s *Store) Init() error {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       s.opt.ConnStr, // DSN data source name
		DefaultStringSize:         256,           // string 类型字段的默认长度
		DisableDatetimePrecision:  true,          // 禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
		DontSupportRenameIndex:    true,          // 重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
		DontSupportRenameColumn:   true,          // 用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
		SkipInitializeWithVersion: false,         // 根据当前 MySQL 版本自动配置
	}), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Warn)})
	if err != nil {
		return fmt.Errorf("connect client failed: %w", err)
	}
	sqlDB, _ := db.DB()
	// SetMaxIdleCons 设置连接池中的最大闲置连接数。
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenCons 设置数据库的最大连接数量。
	sqlDB.SetMaxOpenConns(500)
	// SetConnMaxLifetiment 设置连接的最大可复用时间。
	sqlDB.SetConnMaxLifetime(time.Hour)
	s.db = db
	//自动建表
	s.db.Table(s.opt.Prefix + "_dag").AutoMigrate(&entity.Dag{})
	s.db.Table(s.opt.Prefix + "_task").AutoMigrate(&entity.Task{})
	s.db.Table(s.opt.Prefix + "_dag_instance").AutoMigrate(&entity.DagInstance{})
	s.db.Table(s.opt.Prefix + "_task_instance").AutoMigrate(&entity.TaskInstance{})
	return nil
}

// Close component when we not use it anymore
func (s *Store) Close() {

}

// CreateDag
func (s *Store) CreateDag(dag *entity.Dag) error {
	fmt.Println("CreateDag ", dag)
	_, err := mod.BuildRootNode(mod.MapTasksToGetter(dag.Tasks))
	if err != nil {
		return err
	}
	baseInfo := dag.GetBaseInfo()
	baseInfo.Initial()
	err = s.db.Table(s.opt.Prefix + "_dag").Create(&dag).Error
	if err != nil {
		return fmt.Errorf("insert Dag failed: %w", err)
	}
	for _, task := range dag.Tasks {
		task.DagID = dag.ID
		err = s.db.Table(s.opt.Prefix + "_task").Create(&task).Error
		if err != nil {
			return fmt.Errorf("insert Task failed: %w", err)
		}
	}
	return nil
}

// CreateDagIns
func (s *Store) CreateDagIns(dagIns *entity.DagInstance) error {
	baseInfo := dagIns.GetBaseInfo()
	baseInfo.Initial()
	err := s.db.Table(s.opt.Prefix + "_dag_instance").Create(&dagIns).Error
	if err != nil {
		return fmt.Errorf("insert DagInstance failed: %w", err)
	}
	return nil
}

// CreateTaskIns
func (s *Store) CreateTaskIns(taskIns *entity.TaskInstance) error {
	baseInfo := taskIns.GetBaseInfo()
	baseInfo.Initial()
	err := s.db.Table(s.opt.Prefix + "_task_instance").Create(&taskIns).Error
	if err != nil {
		return fmt.Errorf("insert TaskInstance failed: %w", err)
	}
	return nil
}

// BatchCreatTaskIns
func (s *Store) BatchCreatTaskIns(taskIns []*entity.TaskInstance) error {
	for i := range taskIns {
		baseInfo := taskIns[i].GetBaseInfo()
		baseInfo.Initial()
		err := s.db.Table(s.opt.Prefix + "_task_instance").Create(&taskIns[i]).Error
		if err != nil {
			return fmt.Errorf("insert TaskInstance failed: %w", err)
		}
	}
	return nil
}

// PatchTaskIns
func (s *Store) PatchTaskIns(taskIns *entity.TaskInstance) error {
	if taskIns.ID == "" {
		return fmt.Errorf("id cannot be empty")
	}
	taskIns.Update()
	err := s.db.Table(s.opt.Prefix+"_task_instance").Where("id = ?", taskIns.ID).Updates(&taskIns).Error
	if err != nil {
		return fmt.Errorf("patch TaskInstance failed: %w", err)
	}
	return nil
}

// PatchDagIns
func (s *Store) PatchDagIns(dagIns *entity.DagInstance, mustsPatchFields ...string) error {
	dagIns.Update()
	err := s.db.Table(s.opt.Prefix+"_dag_instance").Where("id = ?", dagIns.ID).Updates(&dagIns).Error
	if err != nil {
		return fmt.Errorf("patch DagInstance failed: %w", err)
	}
	goevent.Publish(&event.DagInstancePatched{
		Payload:         dagIns,
		MustPatchFields: mustsPatchFields,
	})
	return nil
}

// UpdateDag
func (s *Store) UpdateDag(dag *entity.Dag) error {
	// check task's connection
	_, err := mod.BuildRootNode(mod.MapTasksToGetter(dag.Tasks))
	if err != nil {
		return err
	}
	dag.Update()
	err = s.db.Table(s.opt.Prefix+"_dag").Where("id = ?", dag.ID).Updates(&dag).Error
	if err != nil {
		return fmt.Errorf("patch Dag failed: %w", err)
	}
	oldTasks := []entity.Task{}
	err = s.db.Table(s.opt.Prefix+"_task").Where("dag_id = ?", dag.ID).Find(&oldTasks).Error
	if err != nil {
		return fmt.Errorf("get task failed: %w", err)
	}
	for _, task := range dag.Tasks {
		task.DagID = dag.ID
		exits := false
		for _, oldTask := range oldTasks {
			if task.ID == oldTask.ID {
				exits = true
			}
		}
		if exits {
			err = s.db.Table(s.opt.Prefix+"_task").Where("id = ?", task.ID).Updates(&task).Error
			if err != nil {
				return fmt.Errorf("update Task failed: %w", err)
			}
		} else {
			err = s.db.Table(s.opt.Prefix + "_task").Create(&task).Error
			if err != nil {
				return fmt.Errorf("insert Task failed: %w", err)
			}
		}
	}
	for _, oldTask := range oldTasks {
		exits := false
		for _, task := range dag.Tasks {
			if task.ID == oldTask.ID {
				exits = true
			}
		}
		if !exits {
			err = s.db.Table(s.opt.Prefix + "_task").Unscoped().Delete(&oldTask).Error
			if err != nil {
				return fmt.Errorf("delete Task failed: %w", err)
			}
		}
	}
	return nil
}

// UpdateDagIns
func (s *Store) UpdateDagIns(dagIns *entity.DagInstance) error {
	dagIns.Update()
	err := s.db.Table(s.opt.Prefix+"_dag_instance").Where("id = ?", dagIns.ID).Updates(&dagIns).Error
	if err != nil {
		return fmt.Errorf("patch DagInstance failed: %w", err)
	}
	goevent.Publish(&event.DagInstanceUpdated{Payload: dagIns})
	return nil
}

// UpdateTaskIns
func (s *Store) UpdateTaskIns(taskIns *entity.TaskInstance) error {
	taskIns.Update()
	err := s.db.Table(s.opt.Prefix+"_task_instance").Where("id = ?", taskIns.ID).Updates(&taskIns).Error
	if err != nil {
		return fmt.Errorf("patch DagInstance failed: %w", err)
	}
	return nil
}

// BatchUpdateDagIns
func (s *Store) BatchUpdateDagIns(dagIns []*entity.DagInstance) error {
	for i := range dagIns {
		dagIns[i].Update()
		err := s.db.Table(s.opt.Prefix+"_dag_instance").Where("id = ?", dagIns[i].ID).Updates(&dagIns[i]).Error
		if err != nil {
			return fmt.Errorf("patch DagInstance failed: %w", err)
		}
	}
	return nil
}

// BatchUpdateTaskIns
func (s *Store) BatchUpdateTaskIns(taskIns []*entity.TaskInstance) error {
	for i := range taskIns {
		taskIns[i].Update()
		err := s.db.Table(s.opt.Prefix+"_task_instance").Where("id = ?", taskIns[i].ID).Updates(&taskIns[i]).Error
		if err != nil {
			return fmt.Errorf("patch TaskInstance failed: %w", err)
		}
	}
	return nil
}

// GetTaskIns
func (s *Store) GetTaskIns(taskInsId string) (*entity.TaskInstance, error) {
	ret := new(entity.TaskInstance)
	err := s.db.Table(s.opt.Prefix+"_task_instance").Where("id = ?", taskInsId).First(&ret).Error
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// GetDag
func (s *Store) GetDag(dagId string) (*entity.Dag, error) {
	dag := new(entity.Dag)
	err := s.db.Table(s.opt.Prefix+"_dag").Where("id = ?", dagId).First(&dag).Error
	if err != nil {
		return nil, err
	}
	task := []entity.Task{}
	err = s.db.Table(s.opt.Prefix+"_task").Where("dag_id = ?", dagId).Find(&task).Error
	if err != nil {
		return nil, err
	}
	dag.Tasks = task
	return dag, nil
}

// GetDagInstance
func (s *Store) GetDagInstance(dagInsId string) (*entity.DagInstance, error) {
	ret := new(entity.DagInstance)
	err := s.db.Table(s.opt.Prefix+"_dag_instance").Where("id = ?", dagInsId).First(&ret).Error
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// ListDag
func (s *Store) ListDag(input *mod.ListDagInput) ([]*entity.Dag, error) {
	var ret []*entity.Dag
	err := s.db.Table(s.opt.Prefix + "_dag").Find(&ret).Error
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// ListDagInstance
func (s *Store) ListDagInstance(input *mod.ListDagInstanceInput) ([]*entity.DagInstance, error) {
	filterExp := []string{}
	filterArgs := []interface{}{}
	if len(input.Status) > 0 {
		filterExp = append(filterExp, "status in (?) ")
		filterArgs = append(filterArgs, input.Status)
	}
	if input.Worker != "" {
		filterExp = append(filterExp, "worker = ? ")
		filterArgs = append(filterArgs, input.Worker)
	}
	if input.UpdatedEnd > 0 {
		filterExp = append(filterExp, "updated_at <= ? ")
		filterArgs = append(filterArgs, input.UpdatedEnd)
	}
	if input.HasCmd {
		filterExp = append(filterExp, "cmd IS NOT NULL ")
	}
	limit := 10
	if input.Limit > 0 {
		limit = int(input.Limit)
	}
	var ret []*entity.DagInstance
	err := s.db.Table(s.opt.Prefix+"_dag_instance").Where(strings.Join(filterExp, " AND "), filterArgs...).Limit(limit).Order("updated_at DESC").Find(&ret).Error
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// ListTaskInstance
func (s *Store) ListTaskInstance(input *mod.ListTaskInstanceInput) ([]*entity.TaskInstance, error) {
	filterExp := []string{}
	filterArgs := []interface{}{}
	if len(input.IDs) > 0 {
		filterExp = append(filterExp, "id in (?) ")
		filterArgs = append(filterArgs, input.IDs)
	}
	if len(input.Status) > 0 {
		filterExp = append(filterExp, "status in (?) ")
		filterArgs = append(filterArgs, input.Status)
	}
	if input.Expired {
		filterExp = append(filterExp, "updated_at <= ? ")
		filterArgs = append(filterArgs, time.Now().Unix()-5)
	}
	if input.DagInsID != "" {
		filterExp = append(filterExp, "dag_ins_id =? ")
		filterArgs = append(filterArgs, input.DagInsID)
	}
	var ret []*entity.TaskInstance
	err := s.db.Table(s.opt.Prefix+"_task_instance").Where(strings.Join(filterExp, " AND "), filterArgs...).Order("updated_at DESC").Find(&ret).Error
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// BatchDeleteDag
func (s *Store) BatchDeleteDag(ids []string) error {
	err := s.db.Table(s.opt.Prefix+"_dag").Delete(&entity.Dag{}, ids).Error
	if err != nil {
		return err
	}
	return nil
}

// BatchDeleteDagIns
func (s *Store) BatchDeleteDagIns(ids []string) error {
	err := s.db.Table(s.opt.Prefix+"_dag_instance").Delete(&entity.DagInstance{}, ids).Error
	if err != nil {
		return err
	}
	return nil
}

// BatchDeleteTaskIns
func (s *Store) BatchDeleteTaskIns(ids []string) error {
	err := s.db.Table(s.opt.Prefix+"_task_instance").Delete(&entity.TaskInstance{}, ids).Error
	if err != nil {
		return err
	}
	return nil
}

// Marshal
func (s *Store) Marshal(obj interface{}) ([]byte, error) {
	return bson.Marshal(obj)
}

// Unmarshal
func (s *Store) Unmarshal(bytes []byte, ptr interface{}) error {
	return bson.Unmarshal(bytes, ptr)
}
