package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/card"
)

// ── Card CRUD ────────────────────────────────────

func (db *DB) CreateCard(c *card.Card) error {
	query := `INSERT INTO cards (card_id, owner_type, owner_id, card_token, pan_last4,
		card_network, card_type, card_form, expiry_month, expiry_year, status,
		default_currency, daily_limit, monthly_limit, single_tx_limit)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, created_at, updated_at`
	return db.QueryRow(query,
		c.CardID, c.OwnerType, c.OwnerID, c.CardToken, c.PANLast4,
		c.CardNetwork, c.CardType, c.CardForm, c.ExpiryMonth, c.ExpiryYear,
		c.Status, c.DefaultCurrency, c.DailyLimit, c.MonthlyLimit, c.SingleTxLimit,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (db *DB) GetCard(cardID string) (*card.Card, error) {
	c := &card.Card{}
	query := `SELECT id, card_id, owner_type, owner_id, card_token, pan_last4,
		card_network, card_type, card_form, expiry_month, expiry_year, status,
		COALESCE(default_currency,'USD'), daily_limit, monthly_limit, single_tx_limit,
		created_at, updated_at FROM cards WHERE card_id = $1`
	err := db.QueryRow(query, cardID).Scan(
		&c.ID, &c.CardID, &c.OwnerType, &c.OwnerID, &c.CardToken, &c.PANLast4,
		&c.CardNetwork, &c.CardType, &c.CardForm, &c.ExpiryMonth, &c.ExpiryYear,
		&c.Status, &c.DefaultCurrency, &c.DailyLimit, &c.MonthlyLimit, &c.SingleTxLimit,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("card not found: %s", cardID)
	}
	return c, err
}

func (db *DB) GetCardByToken(token string) (*card.Card, error) {
	c := &card.Card{}
	query := `SELECT id, card_id, owner_type, owner_id, card_token, pan_last4,
		card_network, card_type, card_form, expiry_month, expiry_year, status,
		COALESCE(default_currency,'USD'), daily_limit, monthly_limit, single_tx_limit,
		created_at, updated_at FROM cards WHERE card_token = $1`
	err := db.QueryRow(query, token).Scan(
		&c.ID, &c.CardID, &c.OwnerType, &c.OwnerID, &c.CardToken, &c.PANLast4,
		&c.CardNetwork, &c.CardType, &c.CardForm, &c.ExpiryMonth, &c.ExpiryYear,
		&c.Status, &c.DefaultCurrency, &c.DailyLimit, &c.MonthlyLimit, &c.SingleTxLimit,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("card not found by token")
	}
	return c, err
}

func (db *DB) ListCardsByOwner(ownerID string) ([]card.Card, error) {
	query := `SELECT id, card_id, owner_type, owner_id, card_token, pan_last4,
		card_network, card_type, card_form, expiry_month, expiry_year, status,
		COALESCE(default_currency,'USD'), daily_limit, monthly_limit, single_tx_limit,
		created_at, updated_at FROM cards WHERE owner_id = $1 ORDER BY created_at DESC`
	rows, err := db.Query(query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []card.Card
	for rows.Next() {
		var c card.Card
		if err := rows.Scan(
			&c.ID, &c.CardID, &c.OwnerType, &c.OwnerID, &c.CardToken, &c.PANLast4,
			&c.CardNetwork, &c.CardType, &c.CardForm, &c.ExpiryMonth, &c.ExpiryYear,
			&c.Status, &c.DefaultCurrency, &c.DailyLimit, &c.MonthlyLimit, &c.SingleTxLimit,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, nil
}

func (db *DB) UpdateCardStatus(cardID string, status card.CardStatus) error {
	query := `UPDATE cards SET status = $1, updated_at = $2 WHERE card_id = $3`
	_, err := db.Exec(query, status, time.Now(), cardID)
	return err
}

// ── Card Authorization ───────────────────────────

func (db *DB) CreateCardAuthorization(auth *card.CardAuthorization) error {
	query := `INSERT INTO card_authorizations (auth_id, card_id, merchant_name, merchant_country,
		merchant_category_code, transaction_amount, transaction_currency, debit_amount,
		debit_currency, fx_rate, fee_amount, status, decline_reason)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at`
	return db.QueryRow(query,
		auth.AuthID, auth.CardID, nullStr(auth.MerchantName), nullStr(auth.MerchantCountry),
		nullStr(auth.MCC), auth.TransactionAmount, auth.TransactionCurrency, auth.DebitAmount,
		auth.DebitCurrency, nullStr(auth.FXRate), auth.FeeAmount, auth.Status,
		nullStr(string(auth.DeclineReason)),
	).Scan(&auth.ID, &auth.CreatedAt, &auth.UpdatedAt)
}

func nullStr(s string) interface{} {
	if s == "" { return nil }
	return s
}

// ── Card Transactions ────────────────────────────

func (db *DB) ListCardTransactions(cardID string, page, pageSize int) ([]card.CardTransaction, int64, error) {
	var total int64
	db.QueryRow(`SELECT COUNT(*) FROM card_transactions WHERE card_id = $1`, cardID).Scan(&total)

	offset := (page - 1) * pageSize
	if offset < 0 { offset = 0 }

	query := `SELECT id, tx_id, COALESCE(auth_id,''), card_id, transaction_amount,
		transaction_currency, debit_amount, debit_currency, COALESCE(fx_rate,''),
		fee_amount, status, settlement_date, COALESCE(receipt_hash,''), created_at, updated_at
		FROM card_transactions WHERE card_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := db.Query(query, cardID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txs []card.CardTransaction
	for rows.Next() {
		var tx card.CardTransaction
		if err := rows.Scan(
			&tx.ID, &tx.TxID, &tx.AuthID, &tx.CardID, &tx.TransactionAmount,
			&tx.TransactionCurrency, &tx.DebitAmount, &tx.DebitCurrency, &tx.FXRate,
			&tx.FeeAmount, &tx.Status, &tx.SettlementDate, &tx.ReceiptHash,
			&tx.CreatedAt, &tx.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		txs = append(txs, tx)
	}
	return txs, total, nil
}

// ── Fee Rules ────────────────────────────────────

func (db *DB) GetFeeRule(scenario, srcCurrency, tgtCurrency string) (*card.FeeRule, error) {
	r := &card.FeeRule{}
	query := `SELECT id, rule_id, scenario, COALESCE(source_currency,''), COALESCE(target_currency,''),
		COALESCE(country,''), COALESCE(card_network,''), percentage_fee, fixed_fee,
		COALESCE(min_fee,0), COALESCE(max_fee,0), COALESCE(risk_level,''), status
		FROM fee_rules WHERE scenario = $1 AND status = 'ACTIVE' LIMIT 1`
	err := db.QueryRow(query, scenario).Scan(
		&r.ID, &r.RuleID, &r.Scenario, &r.SourceCurrency, &r.TargetCurrency,
		&r.Country, &r.CardNetwork, &r.PercentageFee, &r.FixedFee,
		&r.MinFee, &r.MaxFee, &r.RiskLevel, &r.Status,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fee rule not found: %s", scenario)
	}
	return r, err
}
