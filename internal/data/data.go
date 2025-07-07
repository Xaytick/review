package data

import (
	"review/internal/conf"
	"review/internal/data/query"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewReviewRepo, NewDB, NewESClient, NewRedisClient)

// Data .
type Data struct {
	// TODO wrapped database client
	q   *query.Query
	log *log.Helper
	es  *elasticsearch.TypedClient
	rdb *redis.Client
}

// NewData .
func NewData(db *gorm.DB, esClient *elasticsearch.TypedClient, rdb *redis.Client, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
	}
	query.SetDefault(db)
	return &Data{
		q:   query.Use(db),
		log: log.NewHelper(logger),
		es:  esClient,
		rdb: rdb,
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

func NewRedisClient(c *conf.Data) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
	})
}
