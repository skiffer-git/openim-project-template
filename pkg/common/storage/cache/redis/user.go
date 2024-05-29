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

package redis

import (
	"context"
	"github.com/openimsdk/openim-project-template/pkg/common/storage/database"
	"github.com/openimsdk/openim-project-template/pkg/common/storage/model"
	"time"

	"github.com/dtm-labs/rockscache"
	"github.com/openimsdk/openim-project-template/pkg/common/cachekey"
	"github.com/openimsdk/openim-project-template/pkg/common/storage/cache"
	"github.com/redis/go-redis/v9"
)

const (
	userExpireTime = time.Second * 60 * 60 * 12
)

type User struct {
	cache.BatchDeleter
	userDB     database.User
	expireTime time.Duration
	rcClient   *rockscache.Client
}

func NewUser(rdb redis.UniversalClient, userDB database.User, options *rockscache.Options) cache.User {
	batchHandler := NewBatchDeleterRedis(rdb, options, nil)
	return &User{
		BatchDeleter: batchHandler,
		rcClient:     rockscache.NewClient(rdb, *options),
		userDB:       userDB,
		expireTime:   userExpireTime,
	}
}

func (u *User) CloneUserCache() cache.User {
	return &User{
		BatchDeleter: u.BatchDeleter.Clone(),
		rcClient:     u.rcClient,
		userDB:       u.userDB,
		expireTime:   u.expireTime,
	}
}

func (u *User) getUserInfoKey(userID string) string {
	return cachekey.GetUserInfoKey(userID)
}

func (u *User) getUserGlobalRecvMsgOptKey(userID string) string {
	return cachekey.GetUserGlobalRecvMsgOptKey(userID)
}

func (u *User) GetUsersInfo(ctx context.Context, userIDs []string) ([]*model.User, error) {
	return batchGetCache(ctx, u.rcClient, u.expireTime, userIDs, func(userID string) string {
		return u.getUserInfoKey(userID)
	}, func(ctx context.Context, userID string) (*model.User, error) {
		return u.userDB.Take(ctx, userID)
	})
}

func (u *User) DelUsersInfo(userIDs ...string) cache.User {
	keys := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, u.getUserInfoKey(userID))
	}
	cache := u.CloneUserCache()
	cache.AddKeys(keys...)

	return cache
}

type Comparable interface {
	~int | ~string | ~float64 | ~int32
}
