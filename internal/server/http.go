package server

import (
	"net/http"
	ai_v1 "review/api/ai/v1"
	v1 "review/api/review/v1"
	user_v1 "review/api/user/v1"
	"review/internal/conf"
	"review/internal/service"

	"github.com/go-kratos-ecosystem/components/v2/middleware/cors"
	"github.com/go-kratos/kratos/v2/encoding/json"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/encoding/protojson"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, review *service.ReviewService, aiAgent *service.AIAgentService, user *service.UserService, logger log.Logger) *kratoshttp.Server {
	json.MarshalOptions = protojson.MarshalOptions{
		EmitUnpopulated: true,
	}

	var opts = []kratoshttp.ServerOption{
		kratoshttp.Middleware(
			recovery.Recovery(),
			cors.Cors(
				cors.AllowedOrigins("*"),
				cors.AllowedMethods("GET", "POST", "PUT", "DELETE", "OPTIONS"),
				cors.AllowedHeaders("Content-Type", "Authorization"),
			),
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
	ai_v1.RegisterAIAgentHTTPServer(srv, aiAgent)
	user_v1.RegisterUserHTTPServer(srv, user)

	srv.HandlePrefix("/user/", http.StripPrefix("/user/", http.FileServer(http.Dir("../../frontend/user"))))
	srv.HandlePrefix("/ai-agent/", http.StripPrefix("/ai-agent/", http.FileServer(http.Dir("../../frontend/ai-agent"))))
	srv.HandlePrefix("/dashboard/", http.StripPrefix("/dashboard/", http.FileServer(http.Dir("../../frontend/dashboard"))))

	return srv
}
