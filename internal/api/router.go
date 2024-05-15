package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/openimsdk/openim-project-template/pkg/rpcclient"
	"github.com/openimsdk/tools/discovery"
	"github.com/openimsdk/tools/mw"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func newGinRouter(disCov discovery.SvcDiscoveryRegistry, config *Config) *gin.Engine {
	disCov.AddOption(mw.GrpcClient(), grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, "round_robin")))
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("required_if", RequiredIf)
	}
	r.Use(gin.Recovery(), mw.CorsHandler(), mw.GinParseOperationID())
	// init rpc client here
	userRpc := rpcclient.NewUser(disCov, "user")

	u := NewUserApi(*userRpc)
	userRouterGroup := r.Group("/user")
	{
		userRouterGroup.POST("/user_register", u.UserRegister)
		userRouterGroup.POST("/get_users_info", u.GetUsersPublicInfo)
	}
	return r
}
