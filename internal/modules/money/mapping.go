package money

import (
	"time"

	"github.com/SamoySamoy/Soma/internal/modules/money/store"
)

// These helpers turn generated rows into domain values.

func fromGetRow(r store.GetTransactionRow) Transaction {
	return Transaction{
		ID:         r.ID,
		AccountID:  r.AccountID,
		Kind:       r.Kind,
		Amount:     r.AmountMinor,
		OccurredOn: r.OccurredOn.Format(time.DateOnly),
		CategoryID: r.CategoryID,
		Payee:      stringOf(r.Payee),
		Notes:      stringOf(r.Notes),
		TransferID: r.TransferID,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
		Version:    r.Version,
	}
}

func fromListRow(r store.ListTransactionsRow) Transaction {
	return Transaction{
		ID:         r.ID,
		AccountID:  r.AccountID,
		Kind:       r.Kind,
		Amount:     r.AmountMinor,
		OccurredOn: r.OccurredOn.Format(time.DateOnly),
		CategoryID: r.CategoryID,
		Payee:      stringOf(r.Payee),
		Notes:      stringOf(r.Notes),
		TransferID: r.TransferID,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
		Version:    r.Version,
	}
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringOf(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
