package snowflake

import (
	"errors"

	sf "github.com/bwmarrin/snowflake"
)

var (
	ErrInvalidInitParam = errors.New("无效的 start time 或 machine id")
	ErrInvalidFormat    = errors.New("无效的格式")
)

var node *sf.Node

func Init(startTime string, machineID int64) error {
	if len(startTime) == 0 || machineID <= 0 {
		return ErrInvalidInitParam
	}

	// 不自定义epoch，使用库默认值
	var err error
	node, err = sf.NewNode(machineID)
	return err

}

func GenID() int64 {
	return node.Generate().Int64()
}
