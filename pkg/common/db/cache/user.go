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

package cache

import (
	"context"
	"time"

	"github.com/dtm-labs/rockscache"
	"github.com/openimsdk/openim-project-template/pkg/common/cachekey"
	"github.com/openimsdk/openim-project-template/pkg/common/config"
	relationtb "github.com/openimsdk/openim-project-template/pkg/common/db/table/relation"
	"github.com/openimsdk/tools/log"
	"github.com/redis/go-redis/v9"
)

const (
	userExpireTime            = time.Second * 60 * 60 * 12
	olineStatusKey            = "ONLINE_STATUS:"
	userOlineStatusExpireTime = time.Second * 60 * 60 * 24
	statusMod                 = 501
)

type UserCache interface {
	metaCache
	NewCache() UserCache
	GetUserInfo(ctx context.Context, userID string) (userInfo *relationtb.UserModel, err error)
	GetUsersInfo(ctx context.Context, userIDs []string) ([]*relationtb.UserModel, error)
	DelUsersInfo(userIDs ...string) UserCache
}

type UserCacheRedis struct {
	metaCache
	rdb redis.UniversalClient
	// userDB     relationtb.UserModelInterface
	userDB     relationtb.UserModelInterface
	expireTime time.Duration
	rcClient   *rockscache.Client
}

func NewUserCacheRedis(rdb redis.UniversalClient, localCache *config.LocalCache, userDB relationtb.UserModelInterface, options rockscache.Options) UserCache {
	rcClient := rockscache.NewClient(rdb, options)
	mc := NewMetaCacheRedis(rcClient)
	u := localCache.User
	log.ZDebug(context.Background(), "user local cache init", "Topic", u.Topic, "SlotNum", u.SlotNum, "SlotSize", u.SlotSize, "enable", u.Enable())
	mc.SetTopic(u.Topic)
	mc.SetRawRedisClient(rdb)
	return &UserCacheRedis{
		rdb:        rdb,
		metaCache:  NewMetaCacheRedis(rcClient),
		userDB:     userDB,
		expireTime: userExpireTime,
		rcClient:   rcClient,
	}
}

func (u *UserCacheRedis) NewCache() UserCache {
	return &UserCacheRedis{
		rdb:        u.rdb,
		metaCache:  u.Copy(),
		userDB:     u.userDB,
		expireTime: u.expireTime,
		rcClient:   u.rcClient,
	}
}

func (u *UserCacheRedis) getUserInfoKey(userID string) string {
	return cachekey.GetUserInfoKey(userID)
}

func (u *UserCacheRedis) getUserGlobalRecvMsgOptKey(userID string) string {
	return cachekey.GetUserGlobalRecvMsgOptKey(userID)
}

func (u *UserCacheRedis) GetUserInfo(ctx context.Context, userID string) (userInfo *relationtb.UserModel, err error) {
	return getCache(ctx, u.rcClient, u.getUserInfoKey(userID), u.expireTime, func(ctx context.Context) (*relationtb.UserModel, error) {
		return u.userDB.Take(ctx, userID)
	})
}

func (u *UserCacheRedis) GetUsersInfo(ctx context.Context, userIDs []string) ([]*relationtb.UserModel, error) {
	return batchGetCache2(ctx, u.rcClient, u.expireTime, userIDs, func(userID string) string {
		return u.getUserInfoKey(userID)
	}, func(ctx context.Context, userID string) (*relationtb.UserModel, error) {
		return u.userDB.Take(ctx, userID)
	})
}

func (u *UserCacheRedis) DelUsersInfo(userIDs ...string) UserCache {
	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, u.getUserInfoKey(userID))
	}
	cache := u.NewCache()
	cache.AddKeys(keys...)

	return cache
}

func (u *UserCacheRedis) GetUserGlobalRecvMsgOpt(ctx context.Context, userID string) (opt int, err error) {
	return getCache(
		ctx,
		u.rcClient,
		u.getUserGlobalRecvMsgOptKey(userID),
		u.expireTime,
		func(ctx context.Context) (int, error) {
			return u.userDB.GetUserGlobalRecvMsgOpt(ctx, userID)
		},
	)
}

func (u *UserCacheRedis) DelUsersGlobalRecvMsgOpt(userIDs ...string) UserCache {
	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, u.getUserGlobalRecvMsgOptKey(userID))
	}
	cache := u.NewCache()
	cache.AddKeys(keys...)

	return cache
}

type Comparable interface {
	~int | ~string | ~float64 | ~int32
}

func RemoveRepeatedElementsInList[T Comparable](slc []T) []T {
	var result []T
	tempMap := map[T]struct{}{}
	for _, e := range slc {
		if _, found := tempMap[e]; !found {
			tempMap[e] = struct{}{}
			result = append(result, e)
		}
	}

	return result
}
