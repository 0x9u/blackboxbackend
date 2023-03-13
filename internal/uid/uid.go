package uid

import (
	"github.com/asianchinaboi/backendserver/internal/config"
	"github.com/asianchinaboi/backendserver/internal/logger"
	"github.com/bwmarrin/snowflake"
)

var (
	Snowflake *snowflake.Node
)

func init() {
	var err error
	Snowflake, err = snowflake.NewNode(config.Config.Server.SnowflakeNodeID)
	if err != nil {
		logger.Error.Fatalln(err)
	}
}
