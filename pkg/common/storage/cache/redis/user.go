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

	"github.com/openimsdk/openim-project-template/pkg/common/storage/cache"
)

const (
	userExpireTime = time.Second * 60 * 60 * 12
)

type User struct {
	userDB     database.User
	expireTime time.Duration
}

func NewUser(userDB database.User) cache.User {
	return &User{
		userDB:     userDB,
		expireTime: userExpireTime,
	}
}

func (u *User) GetUsersInfo(ctx context.Context, userIDs []string) ([]*model.User, error) {
	userID := userIDs[0]
	r, err := u.userDB.Take(ctx, userID)
	return []*model.User{r}, err
}

type Comparable interface {
	~int | ~string | ~float64 | ~int32
}
