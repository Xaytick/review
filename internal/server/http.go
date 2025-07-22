package server

import (
	"net/http"
	ai_pb "review/api/ai/v1"
	v1 "review/api/review/v1"
	"review/internal/conf"
	"review/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, review *service.ReviewService, aiAgent *service.AIAgentService, logger log.Logger) *kratoshttp.Server {
	var opts = []kratoshttp.ServerOption{
		kratoshttp.Middleware(
			recovery.Recovery(),
			validate.Validator(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, kratoshttp.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, kratoshttp.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, kratoshttp.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := kratoshttp.NewServer(opts...)
	v1.RegisterReviewHTTPServer(srv, review)
	ai_pb.RegisterAIAgentHTTPServer(srv, aiAgent)

	staticHandler := http.FileServer(http.Dir("../../frontend"))
	srv.HandlePrefix("/", http.StripPrefix("/", staticHandler))

	return srv
}
