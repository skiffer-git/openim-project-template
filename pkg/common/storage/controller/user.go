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

package controller

import (
	"context"
	"github.com/openimsdk/openim-project-template/pkg/common/storage/cache"
	"github.com/openimsdk/openim-project-template/pkg/common/storage/database"
	"github.com/openimsdk/openim-project-template/pkg/common/storage/model"
	"github.com/openimsdk/tools/db/tx"
	"github.com/openimsdk/tools/errs"
)

type User interface {
	// FindWithError Get the information of the specified user. If the userID is not found, it will also return an error
	FindWithError(ctx context.Context, userIDs []string) (users []*model.User, err error) //1
	// Create Insert multiple external guarantees that the userID is not repeated and does not exist in the storage
	Create(ctx context.Context, users []*model.User) (err error) //1

}

type UserStorageManager struct {
	tx    tx.Tx
	db    database.User
	cache cache.User
}

func NewUser(userDB database.User, cache cache.User, tx tx.Tx) User {
	return &UserStorageManager{db: userDB, cache: cache, tx: tx}
}

// FindWithError Get the information of the specified user and return an error if the userID is not found.
func (u *UserStorageManager) FindWithError(ctx context.Context, userIDs []string) (users []*model.User, err error) {
	users, err = u.cache.GetUsersInfo(ctx, userIDs)
	if err != nil {
		return
	}
	if len(users) != len(userIDs) {
		err = errs.ErrRecordNotFound.WrapMsg("userID not found")
	}
	return
}

// Create Insert multiple external guarantees that the userID is not repeated and does not exist in the storage.
func (u *UserStorageManager) Create(ctx context.Context, users []*model.User) (err error) {
	return u.db.Create(ctx, users)
}
