// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package user

import (
	"context"
	"github.com/openimsdk/openim-project-template/pkg/common/config"
	"github.com/openimsdk/openim-project-template/pkg/common/webhook"
	"github.com/openimsdk/tools/db/redisutil"

	"strings"
	"time"

	"github.com/openimsdk/openim-project-template/pkg/common/convert"
	"github.com/openimsdk/openim-project-template/pkg/common/db/cache"
	"github.com/openimsdk/openim-project-template/pkg/common/db/controller"
	"github.com/openimsdk/openim-project-template/pkg/common/db/mgo"
	tablerelation "github.com/openimsdk/openim-project-template/pkg/common/db/table/relation"
	"github.com/openimsdk/openim-project-template/pkg/common/servererrs"
	pbuser "github.com/openimsdk/openim-project-template/pkg/protocol/user"
	"github.com/openimsdk/openim-project-template/pkg/rpcclient"
	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/tools/db/mongoutil"
	registry "github.com/openimsdk/tools/discovery"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
	"google.golang.org/grpc"
)

type userServer struct {
	db                     controller.UserDatabase
	userNotificationSender *UserNotificationSender
	friendRpcClient        *rpcclient.FriendRpcClient
	groupRpcClient         *rpcclient.GroupRpcClient
	RegisterCenter         registry.SvcDiscoveryRegistry
	config                 *Config
	webhookClient          *webhook.Client
}

type Config struct {
	RpcConfig          config.User
	RedisConfig        config.Redis
	MongodbConfig      config.Mongo
	NotificationConfig config.Notification
	Share              config.Share
	WebhooksConfig     config.Webhooks
	LocalCacheConfig   config.LocalCache
	Discovery          config.Discovery
}

func Start(ctx context.Context, config *Config, client registry.SvcDiscoveryRegistry, server *grpc.Server) error {
	mgocli, err := mongoutil.NewMongoDB(ctx, config.MongodbConfig.Build())
	if err != nil {
		return err
	}
	rdb, err := redisutil.NewRedisClient(ctx, config.RedisConfig.Build())
	if err != nil {
		return err
	}
	users := make([]*tablerelation.UserModel, 0)

	for _, v := range config.Share.IMAdminUserID {
		users = append(users, &tablerelation.UserModel{UserID: v, Nickname: v, AppMangerLevel: constant.AppNotificationAdmin})
	}
	userDB, err := mgo.NewUserMongo(mgocli.GetDB())
	if err != nil {
		return err
	}
	userCache := cache.NewUserCacheRedis(rdb, &config.LocalCacheConfig, userDB, cache.GetDefaultOpt())
	userMongoDB := mgo.NewUserMongoDriver(mgocli.GetDB())
	database := controller.NewUserDatabase(userDB, userCache, mgocli.GetTx(), userMongoDB)
	friendRpcClient := rpcclient.NewFriendRpcClient(client, config.Share.RpcRegisterName.Friend)
	groupRpcClient := rpcclient.NewGroupRpcClient(client, config.Share.RpcRegisterName.Group)
	msgRpcClient := rpcclient.NewMessageRpcClient(client, config.Share.RpcRegisterName.Msg)
	cache.InitLocalCache(&config.LocalCacheConfig)
	u := &userServer{
		db:                     database,
		RegisterCenter:         client,
		friendRpcClient:        &friendRpcClient,
		groupRpcClient:         &groupRpcClient,
		userNotificationSender: NewUserNotificationSender(config, &msgRpcClient, WithUserFunc(database.FindWithError)),
		config:                 config,
		webhookClient:          webhook.NewWebhookClient(config.WebhooksConfig.URL),
	}
	pbuser.RegisterUserServer(server, u)
	return u.db.InitOnce(context.Background(), users)
}

func (s *userServer) GetDesignateUsers(ctx context.Context, req *pbuser.GetDesignateUsersReq) (resp *pbuser.GetDesignateUsersResp, err error) {
	resp = &pbuser.GetDesignateUsersResp{}
	users, err := s.db.FindWithError(ctx, req.UserIDs)
	if err != nil {
		return nil, err
	}
	resp.UsersInfo = convert.UsersDB2Pb(users)
	return resp, nil
}

func (s *userServer) UserRegister(ctx context.Context, req *pbuser.UserRegisterReq) (resp *pbuser.UserRegisterResp, err error) {
	resp = &pbuser.UserRegisterResp{}
	if len(req.Users) == 0 {
		return nil, errs.ErrArgs.WrapMsg("users is empty")
	}
	if req.Secret != s.config.Share.Secret {
		log.ZDebug(ctx, "UserRegister", s.config.Share.Secret, req.Secret)
		return nil, errs.ErrNoPermission.WrapMsg("secret invalid")
	}
	if datautil.DuplicateAny(req.Users, func(e *pbuser.UserInfo) string { return e.UserID }) {
		return nil, errs.ErrArgs.WrapMsg("userID repeated")
	}
	userIDs := make([]string, 0)
	for _, user := range req.Users {
		if user.UserID == "" {
			return nil, errs.ErrArgs.WrapMsg("userID is empty")
		}
		if strings.Contains(user.UserID, ":") {
			return nil, errs.ErrArgs.WrapMsg("userID contains ':' is invalid userID")
		}
		userIDs = append(userIDs, user.UserID)
	}
	exist, err := s.db.IsExist(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	if exist {
		return nil, servererrs.ErrRegisteredAlready.WrapMsg("userID registered already")
	}
	if err := s.webhookBeforeUserRegister(ctx, &s.config.WebhooksConfig.BeforeUserRegister, req); err != nil {
		return nil, err
	}
	now := time.Now()
	users := make([]*tablerelation.UserModel, 0, len(req.Users))
	for _, user := range req.Users {
		users = append(users, &tablerelation.UserModel{
			UserID:           user.UserID,
			Nickname:         user.Nickname,
			FaceURL:          user.FaceURL,
			Ex:               user.Ex,
			CreateTime:       now,
			AppMangerLevel:   user.AppMangerLevel,
			GlobalRecvMsgOpt: user.GlobalRecvMsgOpt,
		})
	}
	if err := s.db.Create(ctx, users); err != nil {
		return nil, err
	}

	s.webhookAfterUserRegister(ctx, &s.config.WebhooksConfig.AfterUserRegister, req)
	return resp, nil
}
