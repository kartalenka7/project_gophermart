package storage

import (
	"context"
	"testing"
	"time"

	"github.com/kartalenka7/project_gophermart/internal/config"
	"github.com/kartalenka7/project_gophermart/internal/logger"
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

	log := logger.InitLog()
	cfg, err := config.GetConfig(log)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	storage, err := NewStorage(ctx, cfg.Database, log)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := storage.GetBalance(context.Background(), tt.login)
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

	log := logger.InitLog()
	cfg, err := config.GetConfig(log)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	storage, err := NewStorage(ctx, cfg.Database, log)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := storage.GetWithdrawals(context.Background(), tt.login)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBStruct.GetWithdrawals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
