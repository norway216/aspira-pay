package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/payment"
)

// CreatePaymentOrder inserts a new payment order.
func (db *DB) CreatePaymentOrder(o *payment.Order) error {
	query := `
		INSERT INTO payment_orders (payment_id, request_id, sender_user_id, receiver_user_id,
			source_currency, target_currency, source_amount, target_amount, fee_amount,
			fx_rate, status, risk_score, quote_id, purpose, country_from, country_to)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at, updated_at`
	return db.QueryRow(query,
		o.PaymentID, o.RequestID, o.SenderUserID, o.ReceiverUserID,
		o.SourceCurrency, o.TargetCurrency, o.SourceAmount, o.TargetAmount, o.FeeAmount,
		o.FXRate, o.Status, o.RiskScore, o.QuoteID,
		o.Purpose, o.CountryFrom, o.CountryTo,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
}

// GetPaymentOrder retrieves a payment by payment_id.
func (db *DB) GetPaymentOrder(paymentID string) (*payment.Order, error) {
	o := &payment.Order{}
	query := `SELECT id, payment_id, request_id, sender_user_id, receiver_user_id,
		source_currency, target_currency, source_amount, target_amount, fee_amount,
		fx_rate, status, COALESCE(risk_score, 0), COALESCE(risk_reasons, ''),
		COALESCE(quote_id, ''), COALESCE(chain_tx_id, ''),
		COALESCE(purpose, ''), COALESCE(country_from, ''), COALESCE(country_to, ''),
		created_at, updated_at
		FROM payment_orders WHERE payment_id = $1`
	err := db.QueryRow(query, paymentID).Scan(
		&o.ID, &o.PaymentID, &o.RequestID, &o.SenderUserID, &o.ReceiverUserID,
		&o.SourceCurrency, &o.TargetCurrency, &o.SourceAmount, &o.TargetAmount, &o.FeeAmount,
		&o.FXRate, &o.Status, &o.RiskScore, &o.RiskReasons,
		&o.QuoteID, &o.ChainTxID,
		&o.Purpose, &o.CountryFrom, &o.CountryTo,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found: %s", paymentID)
	}
	return o, err
}

// GetPaymentByRequestID retrieves a payment by idempotency request_id.
func (db *DB) GetPaymentByRequestID(requestID string) (*payment.Order, error) {
	o := &payment.Order{}
	query := `SELECT id, payment_id, request_id, sender_user_id, receiver_user_id,
		source_currency, target_currency, source_amount, target_amount, fee_amount,
		fx_rate, status, COALESCE(risk_score, 0), COALESCE(risk_reasons, ''),
		COALESCE(quote_id, ''), COALESCE(chain_tx_id, ''),
		COALESCE(purpose, ''), COALESCE(country_from, ''), COALESCE(country_to, ''),
		created_at, updated_at
		FROM payment_orders WHERE request_id = $1`
	err := db.QueryRow(query, requestID).Scan(
		&o.ID, &o.PaymentID, &o.RequestID, &o.SenderUserID, &o.ReceiverUserID,
		&o.SourceCurrency, &o.TargetCurrency, &o.SourceAmount, &o.TargetAmount, &o.FeeAmount,
		&o.FXRate, &o.Status, &o.RiskScore, &o.RiskReasons,
		&o.QuoteID, &o.ChainTxID,
		&o.Purpose, &o.CountryFrom, &o.CountryTo,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error for idempotency check
	}
	return o, err
}

// UpdatePaymentStatus updates payment status with transition validation.
func (db *DB) UpdatePaymentStatus(paymentID string, newStatus payment.PaymentStatus) error {
	current, err := db.GetPaymentOrder(paymentID)
	if err != nil {
		return err
	}
	if !payment.CanTransition(current.Status, newStatus) {
		return fmt.Errorf("invalid payment transition: %s -> %s", current.Status, newStatus)
	}

	query := `UPDATE payment_orders SET status = $1, updated_at = $2 WHERE payment_id = $3`
	_, err = db.Exec(query, newStatus, time.Now(), paymentID)
	return err
}

// UpdatePaymentStatusValidated updates status only if current status matches expected.
func (db *DB) UpdatePaymentStatusValidated(paymentID string, from, to payment.PaymentStatus) error {
	query := `UPDATE payment_orders SET status = $1, updated_at = $2 WHERE payment_id = $3 AND status = $4`
	result, err := db.Exec(query, to, time.Now(), paymentID, from)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("payment state mismatch: expected %s for %s", from, paymentID)
	}
	return nil
}

// UpdatePaymentChainTx updates the chain transaction ID.
func (db *DB) UpdatePaymentChainTx(paymentID, chainTxID string) error {
	query := `UPDATE payment_orders SET chain_tx_id = $1, updated_at = now() WHERE payment_id = $2`
	_, err := db.Exec(query, chainTxID, paymentID)
	return err
}

// UpdatePaymentRisk updates risk assessment results.
func (db *DB) UpdatePaymentRisk(paymentID string, riskScore int, riskReasons string) error {
	query := `UPDATE payment_orders SET risk_score = $1, risk_reasons = $2, updated_at = now() WHERE payment_id = $3`
	_, err := db.Exec(query, riskScore, riskReasons, paymentID)
	return err
}

// ListPaymentOrders returns a paginated, filterable list of payments.
func (db *DB) ListPaymentOrders(q payment.ListQuery) ([]payment.Order, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if q.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, q.Status)
		argIdx++
	}
	if q.SenderID != "" {
		where += fmt.Sprintf(" AND sender_user_id = $%d", argIdx)
		args = append(args, q.SenderID)
		argIdx++
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM payment_orders " + where
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
	offset := (q.Page - 1) * q.PageSize

	selectQuery := `SELECT id, payment_id, request_id, sender_user_id, receiver_user_id,
		source_currency, target_currency, source_amount, target_amount, fee_amount,
		fx_rate, status, COALESCE(risk_score, 0), COALESCE(risk_reasons, ''),
		COALESCE(quote_id, ''), COALESCE(chain_tx_id, ''),
		COALESCE(purpose, ''), COALESCE(country_from, ''), COALESCE(country_to, ''),
		created_at, updated_at
		FROM payment_orders ` + where + ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)
	args = append(args, q.PageSize, offset)

	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []payment.Order
	for rows.Next() {
		var o payment.Order
		if err := rows.Scan(
			&o.ID, &o.PaymentID, &o.RequestID, &o.SenderUserID, &o.ReceiverUserID,
			&o.SourceCurrency, &o.TargetCurrency, &o.SourceAmount, &o.TargetAmount, &o.FeeAmount,
			&o.FXRate, &o.Status, &o.RiskScore, &o.RiskReasons,
			&o.QuoteID, &o.ChainTxID,
			&o.Purpose, &o.CountryFrom, &o.CountryTo,
			&o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, nil
}

// GetDailyTotal returns the total amount for a user on the current day.
func (db *DB) GetDailyTotal(userID string) (int64, error) {
	var total sql.NullInt64
	query := `SELECT SUM(source_amount) FROM payment_orders
		WHERE sender_user_id = $1 AND created_at::date = CURRENT_DATE
		AND status NOT IN ('REJECTED', 'CANCELLED', 'FAILED')`
	err := db.QueryRow(query, userID).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total.Int64, nil
}

// GetRecentTxCount returns the number of transactions in the last N seconds for a user.
func (db *DB) GetRecentTxCount(userID string, seconds int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM payment_orders
		WHERE sender_user_id = $1 AND created_at > now() - ($2 || ' seconds')::INTERVAL`
	err := db.QueryRow(query, userID, fmt.Sprintf("%d", seconds)).Scan(&count)
	return count, err
}
