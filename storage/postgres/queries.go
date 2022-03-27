package postgres

const (
	_insertPaymentInfo = `INSERT INTO payment_details (
ref_id,
name, 
surname, 
email, 
phone, 
address_line_1, 
address_line_2, 
postcode, 
card_number,
currency, 
amount, 
payment_type, 
status,
status_reason)
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) 
ON CONFLICT DO NOTHING;`

	_getPaymentInfo = `
SELECT 
ref_id,
name, 
surname, 
email, 
phone, 
address_line_1, 
address_line_2, 
postcode, 
card_number,
currency, 
amount, 
payment_type, 
status,
status_reason,
updated_timestamp,
insert_timestamp
FROM payment_details 
WHERE ref_id = $1 
LIMIT 1
`
)
