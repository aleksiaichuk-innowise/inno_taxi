package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
)

func (s *WalletService) GetWallet(ctx context.Context, userID string) (service_dto.Wallet, error) {
	return s.repo.GetWalletByUserID(ctx, userID)
}
