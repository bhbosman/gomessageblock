package gomessageblock_test

import (
	"context"
	"github.com/bhbosman/gomessageblock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"testing"
)

func TestWriterFactoryService(t *testing.T) {
	var sut gomessageblock.IReaderWriterFactoryService
	app := fxtest.New(
		t,
		fx.Provide(func() *zap.Logger {
			return zap.NewExample()
		}),
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger}
		}),
		gomessageblock.ProvideReaderWriterFactoryService(),
		fx.Populate(&sut),
	)
	require.NoError(t, app.Err())
	require.NoError(t, app.Start(context.TODO()))
	defer func() {
		require.NoError(t, app.Stop(context.TODO()))
	}()
	controller := gomock.NewController(t)
	defer controller.Finish()
}
