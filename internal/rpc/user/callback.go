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
	cbapi "github.com/openimsdk/openim-project-template/pkg/callbackstruct"
	"github.com/openimsdk/openim-project-template/pkg/common/config"
	"github.com/openimsdk/openim-project-template/pkg/common/webhook"
	pbuser "github.com/openimsdk/openim-project-template/pkg/protocol/user"
)

func (s *userServer) webhookBeforeUserRegister(ctx context.Context, before *config.BeforeConfig, req *pbuser.UserRegisterReq) error {
	return webhook.WithCondition(ctx, before, func(ctx context.Context) error {
		cbReq := &cbapi.CallbackBeforeUserRegisterReq{
			CallbackCommand: cbapi.CallbackBeforeUserRegisterCommand,
			Secret:          req.Secret,
			Users:           req.Users,
		}

		resp := &cbapi.CallbackBeforeUserRegisterResp{}

		if err := s.webhookClient.SyncPost(ctx, cbReq.GetCallbackCommand(), cbReq, resp, before); err != nil {
			return err
		}

		if len(resp.Users) != 0 {
			req.Users = resp.Users
		}
		return nil
	})
}

func (s *userServer) webhookAfterUserRegister(ctx context.Context, after *config.AfterConfig, req *pbuser.UserRegisterReq) {
	cbReq := &cbapi.CallbackAfterUserRegisterReq{
		CallbackCommand: cbapi.CallbackAfterUserRegisterCommand,
		Secret:          req.Secret,
		Users:           req.Users,
	}

	s.webhookClient.AsyncPost(ctx, cbReq.GetCallbackCommand(), cbReq, &cbapi.CallbackAfterUserRegisterResp{}, after)
}
