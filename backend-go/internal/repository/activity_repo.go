package repository

import "time"

func (db *DB) InsertActivity(activityID, userID, activityType, refType, refID, title, subtitle string, amount int64, currency, status string) {
	db.Exec(`INSERT INTO activity_feed (activity_id, user_id, activity_type, ref_type, ref_id, title, subtitle, amount, currency, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		activityID, userID, activityType, refType, refID, title, subtitle, amount, currency, status)
}

func (db *DB) ListUserActivity(userID string, page, pageSize int) ([]map[string]interface{}, int64, error) {
	var total int64
	db.QueryRow(`SELECT COUNT(*) FROM activity_feed WHERE user_id=$1`, userID).Scan(&total)
	offset := (page - 1) * pageSize
	if offset < 0 { offset = 0 }
	rows, err := db.Query(`SELECT activity_id, user_id, activity_type, ref_type, ref_id, title, COALESCE(subtitle,''), amount, COALESCE(currency,''), COALESCE(status,''), created_at
		FROM activity_feed WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, pageSize, offset)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var results []map[string]interface{}
	for rows.Next() {
		var id, uid, at, rt, ri, title, sub, cur, st string
		var amt int64
		var ca time.Time
		rows.Scan(&id, &uid, &at, &rt, &ri, &title, &sub, &amt, &cur, &st, &ca)
		results = append(results, map[string]interface{}{
			"activity_id": id, "user_id": uid, "activity_type": at, "ref_type": rt,
			"ref_id": ri, "title": title, "subtitle": sub, "amount": amt,
			"currency": cur, "status": st, "created_at": ca,
		})
	}
	return results, total, nil
}

// TakeBalanceSnapshot records current balance state for recovery (§13.4).
func (db *DB) TakeBalanceSnapshot(accountID, currency string) {
	db.Exec(`INSERT INTO account_balance_snapshots (snapshot_id, account_id, currency, available_balance, frozen_balance, ledger_balance, last_ledger_seq)
		SELECT 'snap_'||gen_random_uuid()::text, $1, $2, available_balance, frozen_balance, 0, 0
		FROM accounts WHERE account_id=$1 AND currency=$2`, accountID, currency)
}

// CheckLedgerIdempotency prevents duplicate ledger entries (§12.4).
func (db *DB) CheckLedgerIdempotency(txID, opType string) bool {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM ledger_idempotency WHERE transaction_id=$1 AND operation_type=$2`, txID, opType).Scan(&count)
	return count > 0
}

// RecordLedgerIdempotency marks a ledger operation as completed.
func (db *DB) RecordLedgerIdempotency(txID, opType, batchID string) {
	db.Exec(`INSERT INTO ledger_idempotency (transaction_id, operation_type, ledger_batch_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, txID, opType, batchID)
}
