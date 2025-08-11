package server

import (
	"context"
	"net/http"
	"strings"

	ai_v1 "review/api/ai/v1"
	v1 "review/api/review/v1"
	user_v1 "review/api/user/v1"
	"review/internal/conf"
	"review/internal/service"

	"github.com/go-kratos-ecosystem/components/v2/middleware/cors"
	"github.com/go-kratos/kratos/v2/encoding/json"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"google.golang.org/protobuf/encoding/protojson"
)

// jwtAuthFilter creates a middleware that selectively applies JWT authentication.
func jwtAuthFilter(jwt middleware.Middleware) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			// Whitelist for API routes that do not require JWT authentication.
			whitelist := map[string]bool{
				"/api.user.v1.User/Login":    true,
				"/api.user.v1.User/Register": true,
			}

			if tr, ok := transport.FromServerContext(ctx); ok {
				// Check for static file paths via HTTP transporter
				if httpTr, ok := tr.(kratoshttp.Transporter); ok {
					path := httpTr.Request().URL.Path
					if strings.HasPrefix(path, "/user/") || strings.HasPrefix(path, "/agent/") || strings.HasPrefix(path, "/dashboard/") {
						return handler(ctx, req) // Skip JWT for static files
					}
				}

				// Check for whitelisted API operations
				operation := tr.Operation()
				if _, ok := whitelist[operation]; ok {
					return handler(ctx, req) // Skip JWT for whitelisted API routes
				}
			}

			// For all other routes, apply the JWT middleware.
			return jwt(handler)(ctx, req)
		}
	}
}

// NewClaimsFactory creates a claims factory for JWT middleware.
func NewClaimsFactory() jwtv5.Claims {
	return jwtv5.MapClaims{}
}

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, review *service.ReviewService, agent *service.AgentService, user *service.UserService, logger log.Logger) *kratoshttp.Server {
	json.MarshalOptions = protojson.MarshalOptions{
		EmitUnpopulated: true,
	}

	jwtSecretKey := "your-secret-key"

	// Create the core JWT middleware instance.
	jwtAuth := jwt.Server(
		func(token *jwtv5.Token) (interface{}, error) {
			return []byte(jwtSecretKey), nil
		},
		jwt.WithClaims(NewClaimsFactory),
	)

	var opts = []kratoshttp.ServerOption{
		kratoshttp.Middleware(
			recovery.Recovery(),
			cors.Cors(
				cors.AllowedOrigins("*"),
				cors.AllowedMethods("GET", "POST", "PUT", "DELETE", "OPTIONS"),
				cors.AllowedHeaders("Content-Type", "Authorization"),
			),
			validate.Validator(),
			// Apply our custom filter middleware, which wraps the JWT middleware.
			jwtAuthFilter(jwtAuth),
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
	ai_v1.RegisterAgentServiceHTTPServer(srv, agent)
	user_v1.RegisterUserHTTPServer(srv, user)

	// Static file serving for frontend pages
	srv.HandlePrefix("/user/", http.StripPrefix("/user/", http.FileServer(http.Dir("../../frontend/user"))))
	srv.HandlePrefix("/agent/", http.StripPrefix("/agent/", http.FileServer(http.Dir("../../frontend/agent"))))
	srv.HandlePrefix("/dashboard/", http.StripPrefix("/dashboard/", http.FileServer(http.Dir("../../frontend/dashboard"))))

	return srv
}
