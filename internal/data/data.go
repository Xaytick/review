package data

import (
	"strings"
	"review/internal/conf"
	"review/internal/data/query"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewReviewRepo, NewDB, NewESClient)

// Data .
type Data struct {
	// TODO wrapped database client
	q *query.Query
	log *log.Helper
	es *elasticsearch.TypedClient
}

// NewData .
func NewData(db *gorm.DB, esClient *elasticsearch.TypedClient, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	query.SetDefault(db)
	return &Data{
		q: query.Use(db),
		log: log.NewHelper(logger),
		es: esClient,
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

func NewESClient(c *conf.Elasticsearch) (*elasticsearch.TypedClient, error) {
	cfg := elasticsearch.Config{
		Addresses: c.Addresses,
	}
	return elasticsearch.NewTypedClient(cfg)

}
