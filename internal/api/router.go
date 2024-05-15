package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/openimsdk/openim-project-template/pkg/common/servererrs"
	"github.com/openimsdk/openim-project-template/pkg/rpcclient"
	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/tools/apiresp"
	"github.com/openimsdk/tools/discovery"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/mw"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
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
	userRpc := rpcclient.NewUser(disCov, config.Share.RpcRegisterName.User, config.Share.RpcRegisterName.MessageGateway,
		config.Share.IMAdminUserID)
	authRpc := rpcclient.NewAuth(disCov, config.Share.RpcRegisterName.Auth)

	u := NewUserApi(*userRpc)
	ParseToken := GinParseToken(authRpc)
	userRouterGroup := r.Group("/user")
	{
		userRouterGroup.POST("/user_register", u.UserRegister)
		userRouterGroup.POST("/get_users_info", ParseToken, u.GetUsersPublicInfo)
	}
	return r
}

func GinParseToken(authRPC *rpcclient.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost:
			token := c.Request.Header.Get(constant.Token)
			if token == "" {
				log.ZWarn(c, "header get token error", servererrs.ErrArgs.WrapMsg("header must have token"))
				apiresp.GinError(c, servererrs.ErrArgs.WrapMsg("header must have token"))
				c.Abort()
				return
			}
			resp, err := authRPC.ParseToken(c, token)
			if err != nil {
				//note :just for template can be run
				c.Set(constant.OpUserPlatform, "Admin")
				c.Set(constant.OpUserID, "imAdmin")
				c.Next()
				return
				//apiresp.GinError(c, err)
				//c.Abort()
				//return
			}
			c.Set(constant.OpUserPlatform, constant.PlatformIDToName(int(resp.PlatformID)))
			c.Set(constant.OpUserID, resp.UserID)
			c.Next()
		}
	}
}
