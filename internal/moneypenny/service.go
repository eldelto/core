package moneypenny

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/eldelto/core/internal/util"
)

const georgeDateTimeLayout = "2006-01-02T15:04:05Z0700"

type Amount int

func (a Amount) String() string {
	return fmt.Sprintf("%d.%d", a/100, util.AbsI(a%100))
}

type Transaction struct {
	PartnerName string
	Details     string
	Date        time.Time
	Amount      Amount
	Currency    string
}

type jsonTransaction struct {
	Booking     string `json:"booking"`
	PartnerName string `json:"partnerName"`
	Amount      struct {
		Value     int    `json:"value"`
		Precision int    `json:"precision"`
		Currency  string `json:"currency"`
	} `json:"amount"`
	Reference string `json:"reference"`
}

func toTransaction(jt jsonTransaction) (Transaction, error) {
	date, err := time.Parse(georgeDateTimeLayout, jt.Booking)
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to parse booking date: %w", err)
	}

	return Transaction{
		PartnerName: jt.PartnerName,
		Details:     jt.Reference,
		Date:        date,
		Amount:      Amount(jt.Amount.Value),
		Currency:    jt.Amount.Currency,
	}, nil
}

func isRelevant(partnerName string) bool {
	partnerName = strings.ToLower(partnerName)

	return strings.Contains(partnerName, "spar") ||
		strings.Contains(partnerName, "hofer") ||
		strings.Contains(partnerName, "billa") ||
		strings.Contains(partnerName, "praskac") ||
		strings.Contains(partnerName, "starkl") ||
		strings.Contains(partnerName, "obi") ||
		strings.Contains(partnerName, "hornbach")
}

func ParseJSON(r io.Reader) ([]Transaction, error) {
	var jsonTransactions []jsonTransaction

	if err := json.NewDecoder(r).Decode(&jsonTransactions); err != nil {
		return nil, fmt.Errorf("failed to decode transactions from JSON: %w", err)
	}

	result := []Transaction{}
	for _, jt := range jsonTransactions {
		if !isRelevant(jt.PartnerName) {
			continue
		}

		t, err := toTransaction(jt)
		if err != nil {
			return nil, fmt.Errorf("failed to convert transaction %q: %w",
				jt.Reference, err)
		}

		result = append(result, t)
	}

	return result, nil
}
