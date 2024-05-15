package webhook

import (
	"context"
	"github.com/openimsdk/openim-project-template/pkg/common/config"
)

func WithCondition(ctx context.Context, before *config.BeforeConfig, callback func(context.Context) error) error {
	if !before.Enable {
		return nil
	}
	return callback(ctx)
}
