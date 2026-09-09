package http

import (
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
)

type walletResp struct {
	ID                string `json:"id"`
	UserID            string `json:"user_id"`
	BalanceMinorUnits int64  `json:"balance_minor_units"`
}

func walletRespFrom(w service_dto.Wallet) walletResp {
	return walletResp{ID: w.ID, UserID: w.UserID, BalanceMinorUnits: w.BalanceMinorUnits}
}

type transactionResp struct {
	ID               string `json:"id"`
	WalletID         string `json:"wallet_id"`
	Type             string `json:"type"`
	AmountMinorUnits int64  `json:"amount_minor_units"`
	ReferenceID      string `json:"reference_id"`
}

func transactionRespFrom(t service_dto.Transaction) transactionResp {
	return transactionResp{
		ID:               t.ID,
		WalletID:         t.WalletID,
		Type:             string(t.Type),
		AmountMinorUnits: t.AmountMinorUnits,
		ReferenceID:      t.ReferenceID,
	}
}
