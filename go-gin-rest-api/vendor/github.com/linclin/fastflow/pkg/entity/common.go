package entity

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/linclin/fastflow/store"
)

// BaseInfo
type BaseInfo struct {
	ID        string `yaml:"id" json:"id" bson:"_id" gorm:"primarykey"`
	CreatedAt int64  `yaml:"createdAt" json:"createdAt" bson:"createdAt"`
	UpdatedAt int64  `yaml:"updatedAt" json:"updatedAt" bson:"updatedAt"`
}

// GetBaseInfo getter
func (b *BaseInfo) GetBaseInfo() *BaseInfo {
	return b
}

// Initial base info
func (b *BaseInfo) Initial() {
	if b.ID == "" {
		b.ID = store.NextStringID()
	}
	b.CreatedAt = time.Now().Unix()
	b.UpdatedAt = time.Now().Unix()
}

// Update
func (b *BaseInfo) Update() {
	b.UpdatedAt = time.Now().Unix()
}

// BaseInfoGetter
type BaseInfoGetter interface {
	GetBaseInfo() *BaseInfo
}

type StringArray []string

func (StringArray) GormDataType() string {
	return "json"
}

// 实现 sql.Scanner 接口，Scan 将 value 扫描至 Jsonb
func (d *StringArray) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, d)
}

// 实现 driver.Valuer 接口，Value 返回 json value
func (d StringArray) Value() (driver.Value, error) {
	return json.Marshal(d)
}

type StringMap map[string]interface{}

func (StringMap) GormDataType() string {
	return "json"
}

// 实现 sql.Scanner 接口，Scan 将 value 扫描至 Jsonb
func (d *StringMap) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, d)
}

// 实现 driver.Valuer 接口，Value 返回 json value
func (d StringMap) Value() (driver.Value, error) {
	return json.Marshal(d)
}
