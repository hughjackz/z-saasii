package repository

// serial.go — Certificate serial number tracking (SECC Leaf).

// GetNextSerialNumber returns the next serial number for a tenant's SECC Leaf certs,
// auto-initializing at 0x13155BC (20012476) if no record exists.
func GetNextSerialNumber(tenantID string, certType string) (int64, error) {
	// Ensure tenant record exists; insert with default if not
	_, _ = DB.Exec(
		`INSERT IGNORE INTO cert_serial (tenant_id, cert_type, serial_no) VALUES (?, ?, 20012476)`,
		tenantID, certType)

	// Increment and return
	_, _ = DB.Exec(
		`UPDATE cert_serial SET serial_no = serial_no + 1 WHERE tenant_id = ? AND cert_type = ?`,
		tenantID, certType)

	var serialNo int64
	err := DB.Get(&serialNo,
		`SELECT serial_no FROM cert_serial WHERE tenant_id = ? AND cert_type = ?`,
		tenantID, certType)
	return serialNo, err
}
