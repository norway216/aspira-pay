package repository

import (
	"database/sql"
	"fmt"

	"github.com/aspira/aspira-pay/internal/domain/transfer"
)

// ── Transfer Orders ──────────────────────────────

func (db *DB) CreateTransferOrder(o *transfer.TransferOrder) error {
	query := `INSERT INTO transfer_orders (transfer_id, payer_user_id, payer_account_id, receiver_user_id,
		receiver_account_id, source_currency, target_currency, source_amount, target_amount, fee_amount,
		fx_rate, quote_id, payment_link_id, status, remark, idempotency_key, completed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING id, created_at, updated_at`
	return db.QueryRow(query,
		o.TransferID, o.PayerUserID, o.PayerAccountID, o.ReceiverUserID, o.ReceiverAccountID,
		o.SourceCurrency, o.TargetCurrency, o.SourceAmount, o.TargetAmount, o.FeeAmount,
		nullS(o.FXRate), nullS(o.QuoteID), nullS(o.PaymentLinkID), o.Status, nullS(o.Remark),
		o.IdempotencyKey, o.CompletedAt,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
}

func nullS(s string) interface{} { if s == "" { return nil }; return s }

func (db *DB) UpdateTransferStatus(transferID string, status transfer.TransferStatus) error {
	_, err := db.Exec(`UPDATE transfer_orders SET status=$1, updated_at=now(), completed_at=CASE WHEN $1 IN ('succeeded','failed','rejected') THEN now() ELSE completed_at END WHERE transfer_id=$2`, status, transferID)
	return err
}

func (db *DB) ListTransferOrders(userID string, page, pageSize int) ([]transfer.TransferOrder, int64, error) {
	var total int64
	db.QueryRow(`SELECT COUNT(*) FROM transfer_orders WHERE payer_user_id=$1 OR receiver_user_id=$1`, userID).Scan(&total)
	offset := (page - 1) * pageSize
	if offset < 0 { offset = 0 }
	rows, err := db.Query(`SELECT id, transfer_id, payer_user_id, payer_account_id, receiver_user_id,
		receiver_account_id, source_currency, target_currency, source_amount, target_amount, fee_amount,
		COALESCE(fx_rate,''), COALESCE(quote_id,''), COALESCE(payment_link_id,''), status,
		COALESCE(remark,''), idempotency_key, created_at, updated_at, completed_at
		FROM transfer_orders WHERE payer_user_id=$1 OR receiver_user_id=$1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, pageSize, offset)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var orders []transfer.TransferOrder
	for rows.Next() {
		var o transfer.TransferOrder
		if err := rows.Scan(&o.ID, &o.TransferID, &o.PayerUserID, &o.PayerAccountID, &o.ReceiverUserID,
			&o.ReceiverAccountID, &o.SourceCurrency, &o.TargetCurrency, &o.SourceAmount, &o.TargetAmount,
			&o.FeeAmount, &o.FXRate, &o.QuoteID, &o.PaymentLinkID, &o.Status, &o.Remark,
			&o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt, &o.CompletedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, nil
}

// ── Transfer Contacts ────────────────────────────

func (db *DB) UpsertTransferContact(contactID, ownerID, targetID, targetAcct, displayName, aspiraID, acctMasked, currency string, amount int64) {
	db.Exec(`INSERT INTO transfer_contacts (contact_id, owner_user_id, target_user_id, target_account_id,
		target_display_name, target_aspira_id, target_account_no_masked, target_currency, last_transfer_at, transfer_count, total_amount)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,now(),1,$9)
		ON CONFLICT (owner_user_id, target_user_id, target_account_id)
		DO UPDATE SET last_transfer_at=now(), transfer_count=transfer_contacts.transfer_count+1,
		total_amount=transfer_contacts.total_amount+$9, updated_at=now()`,
		contactID, ownerID, targetID, targetAcct, displayName, aspiraID, acctMasked, currency, amount)
}

func (db *DB) ListTransferContacts(userID string) ([]transfer.TransferContact, error) {
	rows, err := db.Query(`SELECT id, contact_id, owner_user_id, target_user_id, target_account_id,
		COALESCE(target_display_name,''), COALESCE(target_aspira_id,''), COALESCE(target_account_no_masked,''),
		COALESCE(target_currency,''), last_transfer_at, transfer_count, total_amount, status, created_at, updated_at
		FROM transfer_contacts WHERE owner_user_id=$1 AND status='active' ORDER BY last_transfer_at DESC LIMIT 20`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var contacts []transfer.TransferContact
	for rows.Next() {
		var c transfer.TransferContact
		if err := rows.Scan(&c.ID, &c.ContactID, &c.OwnerUserID, &c.TargetUserID, &c.TargetAccountID,
			&c.TargetDisplayName, &c.TargetAspiraID, &c.TargetAccountNoMasked, &c.TargetCurrency,
			&c.LastTransferAt, &c.TransferCount, &c.TotalAmount, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}

// ── Payment Links ────────────────────────────────

func (db *DB) CreatePaymentLink(link *transfer.PaymentLink) error {
	return db.QueryRow(`INSERT INTO payment_links (payment_link_id, link_token_hash, link_token_prefix,
		creator_user_id, receiver_account_id, amount, currency, title, description, expire_at, max_pay_count, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, created_at, updated_at`,
		link.PaymentLinkID, link.LinkTokenHash, link.LinkTokenPrefix, link.CreatorUserID,
		link.ReceiverAccountID, link.Amount, link.Currency, nullS(link.Title), nullS(link.Description),
		link.ExpireAt, link.MaxPayCount, link.Status,
	).Scan(&link.ID, &link.CreatedAt, &link.UpdatedAt)
}

func (db *DB) GetPaymentLinkByHash(tokenHash string) (*transfer.PaymentLink, error) {
	link := &transfer.PaymentLink{}
	err := db.QueryRow(`SELECT id, payment_link_id, link_token_hash, link_token_prefix, creator_user_id,
		receiver_account_id, amount, currency, COALESCE(title,''), COALESCE(description,''), expire_at,
		max_pay_count, paid_count, status, created_at, updated_at, paid_at, cancelled_at
		FROM payment_links WHERE link_token_hash=$1`, tokenHash).Scan(
		&link.ID, &link.PaymentLinkID, &link.LinkTokenHash, &link.LinkTokenPrefix, &link.CreatorUserID,
		&link.ReceiverAccountID, &link.Amount, &link.Currency, &link.Title, &link.Description,
		&link.ExpireAt, &link.MaxPayCount, &link.PaidCount, &link.Status,
		&link.CreatedAt, &link.UpdatedAt, &link.PaidAt, &link.CancelledAt)
	if err == sql.ErrNoRows { return nil, fmt.Errorf("payment link not found") }
	return link, err
}

func (db *DB) GetPaymentLink(linkID string) (*transfer.PaymentLink, error) {
	link := &transfer.PaymentLink{}
	err := db.QueryRow(`SELECT id, payment_link_id, link_token_hash, link_token_prefix, creator_user_id,
		receiver_account_id, amount, currency, COALESCE(title,''), COALESCE(description,''), expire_at,
		max_pay_count, paid_count, status, created_at, updated_at, paid_at, cancelled_at
		FROM payment_links WHERE payment_link_id=$1`, linkID).Scan(
		&link.ID, &link.PaymentLinkID, &link.LinkTokenHash, &link.LinkTokenPrefix, &link.CreatorUserID,
		&link.ReceiverAccountID, &link.Amount, &link.Currency, &link.Title, &link.Description,
		&link.ExpireAt, &link.MaxPayCount, &link.PaidCount, &link.Status,
		&link.CreatedAt, &link.UpdatedAt, &link.PaidAt, &link.CancelledAt)
	if err == sql.ErrNoRows { return nil, fmt.Errorf("payment link not found: %s", linkID) }
	return link, err
}

func (db *DB) UpdatePaymentLinkStatus(linkID string, status transfer.PaymentLinkStatus) error {
	_, err := db.Exec(`UPDATE payment_links SET status=$1, updated_at=now(),
		cancelled_at=CASE WHEN $1='cancelled' THEN now() ELSE cancelled_at END WHERE payment_link_id=$2`, status, linkID)
	return err
}

func (db *DB) UpdatePaymentLinkPaid(linkID string) {
	db.Exec(`UPDATE payment_links SET status='paid', paid_count=paid_count+1, paid_at=now(), updated_at=now() WHERE payment_link_id=$1`, linkID)
}

func (db *DB) ExpirePendingPaymentLinks() (int64, error) {
	r, err := db.Exec(`UPDATE payment_links SET status='expired', updated_at=now() WHERE status='pending' AND expire_at < now()`)
	if err != nil { return 0, err }
	return r.RowsAffected()
}

// List payment links created by a user
func (db *DB) ListPaymentLinksByCreator(userID string) ([]transfer.PaymentLink, error) {
	rows, err := db.Query(`SELECT id, payment_link_id, link_token_hash, link_token_prefix, creator_user_id,
		receiver_account_id, amount, currency, COALESCE(title,''), COALESCE(description,''), expire_at,
		max_pay_count, paid_count, status, created_at, updated_at, paid_at, cancelled_at
		FROM payment_links WHERE creator_user_id=$1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var links []transfer.PaymentLink
	for rows.Next() {
		var l transfer.PaymentLink
		if err := rows.Scan(&l.ID, &l.PaymentLinkID, &l.LinkTokenHash, &l.LinkTokenPrefix, &l.CreatorUserID,
			&l.ReceiverAccountID, &l.Amount, &l.Currency, &l.Title, &l.Description,
			&l.ExpireAt, &l.MaxPayCount, &l.PaidCount, &l.Status,
			&l.CreatedAt, &l.UpdatedAt, &l.PaidAt, &l.CancelledAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, nil
}
