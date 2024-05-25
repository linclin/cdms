package mongo

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/linclin/fastflow/keeper"
	"github.com/linclin/fastflow/pkg/event"
	"github.com/linclin/fastflow/pkg/log"
	"github.com/linclin/fastflow/pkg/mod"
	"github.com/linclin/fastflow/store"
	"github.com/shiningrush/goevent"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const LeaderKey = "leader"

// Keeper mysql implement
type Keeper struct {
	opt              *KeeperOption
	leaderClsName    string
	heartbeatClsName string
	mutexClsName     string

	leaderFlag atomic.Value
	keyNumber  int
	db         *gorm.DB

	wg            sync.WaitGroup
	firstInitWg   sync.WaitGroup
	initCompleted bool
	closeCh       chan struct{}
}

// KeeperOption
type KeeperOption struct {
	// Key the work key, must be the format like "xxxx-{{number}}", number is the code of worker
	Key string
	// mysql connection string
	ConnStr string
	// the prefix will append to the database
	Prefix string
	// UnhealthyTime default 5s, campaign and heartbeat time will be half of it
	UnhealthyTime time.Duration
	// Timeout default 2s
	Timeout time.Duration
}

type Worker struct {
	WorkerKey   string    `gorm:"column:worker_key;unique;comment:WorkerKey"`
	WorkerType  string    `gorm:"column:worker_type;comment:WorkerType"`
	UpdatedAt   time.Time `gorm:"column:updated_at;comment:UpdatedAt"`
	HeartbeatAt time.Time `gorm:"column:heartbeat_at;comment:HeartBeatAt"`
}

func (Worker) TableName() string {
	return "worker"
}

// NewKeeper
func NewKeeper(opt *KeeperOption) *Keeper {
	k := &Keeper{
		opt:     opt,
		closeCh: make(chan struct{}),
	}
	k.leaderFlag.Store(false)
	return k
}

// Init
func (k *Keeper) Init() error {
	if err := k.readOpt(); err != nil {
		return err
	}
	store.InitFlakeGenerator(uint16(k.WorkerNumber()))
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       k.opt.ConnStr, // DSN data source name
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
	k.db = db
	//自动建表
	k.db.Table(k.opt.Prefix + "_worker").AutoMigrate(&Worker{})

	k.firstInitWg.Add(2)
	k.wg.Add(1)
	go k.goElect()
	k.wg.Add(1)
	go k.goHeartBeat()
	k.firstInitWg.Wait()
	k.initCompleted = true
	return nil
}

func (k *Keeper) setLeaderFlag(isLeader bool) {
	k.leaderFlag.Store(isLeader)
	goevent.Publish(&event.LeaderChanged{
		IsLeader:  isLeader,
		WorkerKey: k.WorkerKey(),
	})
}

func (k *Keeper) readOpt() error {
	if k.opt.Key == "" || k.opt.ConnStr == "" {
		return fmt.Errorf("worker key or connection string can not be empty")
	}
	if k.opt.UnhealthyTime == 0 {
		k.opt.UnhealthyTime = time.Second * 5
	}
	if k.opt.Timeout == 0 {
		k.opt.Timeout = time.Second * 2
	}

	number, err := keeper.CheckWorkerKey(k.opt.Key)
	if err != nil {
		return err
	}
	k.keyNumber = number

	k.leaderClsName = "election"
	k.heartbeatClsName = "heartbeat"
	k.mutexClsName = "mutex"
	if k.opt.Prefix != "" {
		k.leaderClsName = fmt.Sprintf("%s_%s", k.opt.Prefix, k.leaderClsName)
		k.heartbeatClsName = fmt.Sprintf("%s_%s", k.opt.Prefix, k.heartbeatClsName)
		k.mutexClsName = fmt.Sprintf("%s_%s", k.opt.Prefix, k.mutexClsName)
	}

	return nil
}

// IsLeader indicate the component if is leader node
func (k *Keeper) IsLeader() bool {
	return k.leaderFlag.Load().(bool)
}

// AliveNodes get all alive nodes
func (k *Keeper) AliveNodes() ([]string, error) {

	var ret []Worker
	err := k.db.Table(k.opt.Prefix + "_worker").Find(&ret).Error
	if err != nil {
		return []string{}, err
	}
	var aliveNodes []string
	for i := range ret {
		aliveNodes = append(aliveNodes, ret[i].WorkerKey)
	}
	return aliveNodes, nil
}

// IsAlive check if a worker still alive
func (k *Keeper) IsAlive(workerKey string) (bool, error) {
	var worker Worker
	k.db.Table(k.opt.Prefix+"_worker").Where("worker_key = ?", workerKey).First(&worker)
	if worker.UpdatedAt.After(time.Now().Add(-1 * k.opt.UnhealthyTime)) {
		return true, nil
	}
	return false, nil
}

// WorkerKey must match `xxxx-1` format
func (k *Keeper) WorkerKey() string {
	return k.opt.Key
}

// WorkerNumber get the the key number of Worker key, if here is a WorkKey like `worker-1`, then it will return "1"
func (k *Keeper) WorkerNumber() int {
	return k.keyNumber
}

// close component
func (k *Keeper) Close() {
	close(k.closeCh)
	k.wg.Wait()
	k.db.Table(k.opt.Prefix+"_worker").Where("worker_key = ?", k.opt.Key).Delete(&Worker{})
}

// this function is just for testing
func (k *Keeper) forceClose() {
	close(k.closeCh)
	k.wg.Wait()
}

func (k *Keeper) goElect() {
	timerCh := time.Tick(k.opt.UnhealthyTime / 2)
	closed := false
	for !closed {
		select {
		case <-k.closeCh:
			closed = true
		case <-timerCh:
			k.elect()
		}
	}
	k.wg.Done()
}

func (k *Keeper) elect() {
	if k.leaderFlag.Load().(bool) {
		if err := k.continueLeader(); err != nil {
			log.Errorf("continue leader failed: %s", err)
			k.setLeaderFlag(false)
			return
		}
	} else {
		if err := k.campaign(); err != nil {
			log.Errorf("campaign failed: %s", err)
			return
		}
	}

	if !k.initCompleted {
		k.firstInitWg.Done()
	}
}

func (k *Keeper) campaign() error {
	var ret []Worker
	err := k.db.Table(k.opt.Prefix + "_worker").Find(&ret).Error
	if err != nil {
		return err
	}
	if len(ret) > 0 {
		if ret[0].WorkerKey == k.opt.Key {
			k.setLeaderFlag(true)
			return nil
		}
		if ret[0].UpdatedAt.Before(time.Now().Add(-1 * k.opt.UnhealthyTime)) {
			leader := Worker{
				WorkerKey:   k.opt.Key,
				WorkerType:  LeaderKey,
				UpdatedAt:   time.Now(),
				HeartbeatAt: time.Now(),
			}
			err := k.db.Table(k.opt.Prefix+"_worker").Where("worker_key = ?", k.opt.Key).Updates(&leader).Error
			if err != nil {
				return fmt.Errorf("update Leader failed: %w", err)
			}
			k.setLeaderFlag(true)
		}
	}
	if len(ret) == 0 {
		leader := Worker{
			WorkerKey:   k.opt.Key,
			WorkerType:  LeaderKey,
			UpdatedAt:   time.Now(),
			HeartbeatAt: time.Now(),
		}
		err := k.db.Table(k.opt.Prefix + "_worker").Create(&leader).Error
		if err != nil {
			log.Errorf("insert campaign rec failed: %s", err)
			return fmt.Errorf("insert failed: %w", err)
		}
		k.setLeaderFlag(true)
	}
	return nil
}

func (k *Keeper) continueLeader() error {
	leader := Worker{
		WorkerKey:   k.opt.Key,
		WorkerType:  LeaderKey,
		UpdatedAt:   time.Now(),
		HeartbeatAt: time.Now(),
	}
	err := k.db.Table(k.opt.Prefix+"_worker").Where("worker_key = ?", k.opt.Key).Updates(&leader).Error
	if err != nil {
		return fmt.Errorf("update Leader failed: %w", err)
	}
	return nil
}

func (k *Keeper) goHeartBeat() {
	timerCh := time.Tick(k.opt.UnhealthyTime / 2)
	closed := false
	for !closed {
		select {
		case <-k.closeCh:
			closed = true
		case <-timerCh:
			if err := k.heartBeat(); err != nil {
				log.Errorf("heart beat failed: %s", err)
				continue
			}
		}
		if !k.initCompleted {
			k.firstInitWg.Done()
		}
	}
	k.wg.Done()
}

func (k *Keeper) heartBeat() error {
	err := k.db.Table(k.opt.Prefix+"_worker").Model(&Worker{}).Where("worker_key = ?", k.opt.Key).Update("updated_at", time.Now()).Error
	if err != nil {
		return fmt.Errorf("update Leader failed: %w", err)
	}
	return nil
}

func boolPtr(b bool) *bool {
	return &b
}

// NewMutex(key string) create a new distributed mutex
func (k *Keeper) NewMutex(key string) mod.DistributedMutex {
	return nil
}
