package initialize

import (
	"fmt"
	"go-gin-rest-api/pkg/global"
	"go-gin-rest-api/pkg/utils"
	"net"

	"github.com/linclin/fastflow"
	mysqlKeeper "github.com/linclin/fastflow/keeper/mysql"
	mysqlStore "github.com/linclin/fastflow/store/mysql"
)

// 初始化流水线
func InitPipeline() {
	net.InterfaceAddrs()
	mysqlDsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?%s",
		global.Conf.Mysql.Username,
		global.Conf.Mysql.Password,
		global.Conf.Mysql.Host,
		global.Conf.Mysql.Port,
		global.Conf.Mysql.Database,
		global.Conf.Mysql.Query,
	)
	localIp, err := utils.GetLocalIP()
	if err != nil {
		panic(fmt.Sprintf("获取本机IP错误: %s", err))
	}
	// init keeper
	keeper := mysqlKeeper.NewKeeper(&mysqlKeeper.KeeperOption{
		Key:     "worker-" + localIp,
		ConnStr: mysqlDsn,
		Prefix:  "pipeline",
	})
	if err := keeper.Init(); err != nil {
		panic(fmt.Sprintf("初始化流水线组件keeper错误: %s", err))
	}
	// init store
	st := mysqlStore.NewStore(&mysqlStore.StoreOption{
		ConnStr: mysqlDsn,
		Prefix:  "pipeline",
	})
	if err := st.Init(); err != nil {
		panic(fmt.Sprintf("初始化流水线组件store错误: %s", err))
	}

	// init fastflow
	if err := fastflow.Init(&fastflow.InitialOption{
		Keeper:            keeper,
		Store:             st,
		ParserWorkersCnt:  10,
		ExecutorWorkerCnt: 50,
		ExecutorTimeout:   600,
	}); err != nil {
		panic(fmt.Sprintf("初始化流水线组件错误: %s", err))
	}
	global.Log.Info("初始化流水线组件完成")
}
