package main

import (
	"flag"
	"review/internal/conf"
	"strings"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
)

var flagconf string

func main() {
	flag.Parse()

	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)

	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	g := gen.NewGenerator(gen.Config{
		OutPath: "../../internal/data/query",
		// 生成全局对象Q和Query接口
		Mode:          gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldNullable: true, // 允许deleted_at为null
	})

	g.UseDB(connectDB(bc.GetData().GetDatabase()))

	g.ApplyBasic(g.GenerateAllTable()...)

	g.Execute()
}

func connectDB(cfg *conf.Data_Database) *gorm.DB {
	if cfg == nil {
		panic("database config is nil")
	}

	switch strings.ToLower(cfg.GetDriver()) {
	case "mysql":
		db, err := gorm.Open(mysql.Open(cfg.GetSource()))
		if err != nil {
			panic(err)
		}
		return db
	default:
		panic("unsupported driver: " + cfg.Driver)
	}
}

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}
