package data

import (
	"strings"
	"review/internal/conf"
	"review/internal/data/query"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewReviewRepo, NewDB)

// Data .
type Data struct {
	// TODO wrapped database client
	q *query.Query
	log *log.Helper
}

// NewData .
func NewData(db *gorm.DB, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	query.SetDefault(db)
	return &Data{
		q: query.Use(db),
		log: log.NewHelper(logger),
	}, cleanup, nil
}

func NewDB(c *conf.Data) (*gorm.DB, error) {
	switch strings.ToLower(c.Database.Driver) {
	case "mysql":
		return gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	default:
		panic("unsupported database driver")
	}
}
