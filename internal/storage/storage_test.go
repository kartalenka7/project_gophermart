package storage

import (
	"context"
	"testing"

	"github.com/kartalenka7/project_gophermart/internal/config"
	"github.com/kartalenka7/project_gophermart/internal/model"
	"github.com/stretchr/testify/require"
)

func TestDBStruct_GetBalance(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		want    model.Balance
		wantErr bool
	}{
		{
			name:    "Get balance test",
			login:   "user2",
			wantErr: false,
		},
	}

	log := config.InitLog()
	cfg, err := config.GetConfig(log)
	require.NoError(t, err)
	storage, err := NewStorage(cfg.Database, cfg.AccrualSys, log)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), model.KeyLogin, tt.login)
			_, err := storage.GetBalance(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBStruct.GetBalance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDBStruct_GetWithdrawals(t *testing.T) {

	tests := []struct {
		name    string
		login   string
		want    []model.OrderWithdraw
		wantErr bool
	}{
		{
			name:    "Get withdrawals test",
			login:   "user2",
			wantErr: false,
		},
	}

	log := config.InitLog()
	cfg, err := config.GetConfig(log)
	require.NoError(t, err)
	storage, err := NewStorage(cfg.Database, cfg.AccrualSys, log)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), model.KeyLogin, tt.login)
			_, err := storage.GetWithdrawals(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBStruct.GetWithdrawals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
