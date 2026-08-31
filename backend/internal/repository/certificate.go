package repository

import (
	"github.com/google/uuid"
	"github.com/yourorg/csms-backend/internal/model"
)

func ListCertificates(callerRole model.Role, callerID string, tenantID string, certType string) ([]*model.Certificate, error) {
	q := `SELECT c.id, c.name, c.cert_group, c.type, c.file_path, c.private_key_path,
		  c.serial_number, c.issuer_name, c.subject_name, c.public_key,
		  c.signature_algorithm, c.hash_algorithm, c.issuer_name_hash, c.issuer_key_hash,
		  c.valid_from, c.valid_to, c.enabled,
		  c.uploaded_at, c.owner_id, c.tenant_id,
		  u.name AS owner_name
		  FROM certificate c LEFT JOIN role u ON u.id = c.owner_id WHERE 1=1`
	args := []interface{}{}
	switch callerRole {
	case model.RoleCSAdmin:
		if tenantID != "" {
			q += " AND c.tenant_id=?"
			args = append(args, tenantID)
		}
	case model.RoleCPOP:
		q += " AND c.tenant_id=?"
		args = append(args, tenantID)
	case model.RoleCPOM:
		q += " AND c.owner_id=?"
		args = append(args, callerID)
	}
	if certType != "" {
		q += " AND c.type=?"
		args = append(args, certType)
	}
	var certs []*model.Certificate
	if err := DB.Select(&certs, q, args...); err != nil {
		return nil, err
	}
	if certs == nil {
		certs = []*model.Certificate{}
	}
	return certs, nil
}

// GetCertByID returns the full certificate record including file paths.
// FindCertByHash looks up a certificate by issuerNameHash and issuerKeyHash.
func FindCertByHash(issuerNameHash, issuerKeyHash string) (bool, error) {
	var count int
	err := DB.Get(&count,
		"SELECT COUNT(*) FROM certificate WHERE issuer_name_hash=? AND issuer_key_hash=?",
		issuerNameHash, issuerKeyHash)
	return count > 0, err
}

// FindCertBySubject looks up a CA certificate whose subject matches the given issuer name.
func FindCertBySubject(issuerName string) (*model.Certificate, error) {
	var c model.Certificate
	err := DB.Get(&c, "SELECT id, name, type, content, subject_name FROM certificate WHERE subject_name=? AND type LIKE '%Root%' LIMIT 1", issuerName)
	return &c, err
}

// GetCertBySerial looks up a certificate by serial number (for PNC Authorize cert validation).
func GetCertBySerial(serialNumber string) (*model.Certificate, error) {
	var c model.Certificate
	err := DB.Get(&c, `SELECT id, name, type, serial_number FROM certificate WHERE serial_number=? LIMIT 1`, serialNumber)
	return &c, err
}

func GetCertByID(id string) (*model.Certificate, error) {
	var c model.Certificate
	err := DB.Get(&c, `SELECT id, name, cert_group, type, file_path, private_key_path,
		owner_id, tenant_id FROM certificate WHERE id=?`, id)
	return &c, err
}

func GetCertificateContent(id string) (string, string, error) {
	var row struct {
		Content    string `db:"content"`
		PrivateKey string `db:"private_key"`
	}
	err := DB.Get(&row, "SELECT content, private_key FROM certificate WHERE id=?", id)
	return row.Content, row.PrivateKey, err
}

func GetCertKeyAndPassphrase(id string) (content, privKey, passphrase string, err error) {
	var row struct {
		Content       string `db:"content"`
		PrivateKey    string `db:"private_key"`
		KeyPassphrase string `db:"key_passphrase"`
	}
	err = DB.Get(&row, "SELECT content, private_key, key_passphrase FROM certificate WHERE id=?", id)
	return row.Content, row.PrivateKey, row.KeyPassphrase, err
}

func GetPrivateKey(id string) (string, error) {
	var pk string
	err := DB.Get(&pk, "SELECT private_key FROM certificate WHERE id=?", id)
	return pk, err
}

func CreateCertificate(c *model.Certificate) error {
	c.ID = uuid.New().String()
	_, err := DB.NamedExec(`INSERT INTO certificate
		(id,name,cert_group,type,content,private_key,key_passphrase,file_path,private_key_path,
		 serial_number,issuer_name,subject_name,public_key,signature_algorithm,
		 hash_algorithm,issuer_name_hash,issuer_key_hash,
		 valid_from,valid_to,enabled,uploaded_at,owner_id,tenant_id)
		VALUES (:id,:name,:cert_group,:type,:content,:private_key,:key_passphrase,:file_path,:private_key_path,
		 :serial_number,:issuer_name,:subject_name,:public_key,:signature_algorithm,
		 :hash_algorithm,:issuer_name_hash,:issuer_key_hash,
		 :valid_from,:valid_to,:enabled,NOW(),:owner_id,:tenant_id)`, c)
	return err
}

func UpdateCertificateName(id, name string) error {
	_, err := DB.Exec("UPDATE certificate SET name=? WHERE id=?", name, id)
	return err
}

func DeleteCertificate(id string) error {
	_, err := DB.Exec("DELETE FROM certificate WHERE id=?", id)
	return err
}

// ─── Contract Cert Group (2.3.2.4.e / 4.2.9.5) ────────────────────────────

// SaveContractCertGroup replaces the stored cert group for a device.
func SaveContractCertGroup(deviceID, tenantID string, group map[string]string) error {
	tx, _ := DB.Begin()
	defer tx.Rollback()
	_, _ = tx.Exec("DELETE FROM contract_cert_group WHERE device_id=?", deviceID)
	for certType, certName := range group {
		_, err := tx.Exec(
			"INSERT INTO contract_cert_group (device_id, tenant_id, cert_type, cert_name) VALUES (?,?,?,?)",
			deviceID, tenantID, certType, certName)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetContractCertGroup returns the stored cert group for a device.
func GetContractCertGroup(deviceID string) (map[string]string, error) {
	rows, err := DB.Query(
		"SELECT cert_type, cert_name FROM contract_cert_group WHERE device_id=?",
		deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	group := make(map[string]string)
	for rows.Next() {
		var ct, cn string
		if err := rows.Scan(&ct, &cn); err != nil {
			return nil, err
		}
		group[ct] = cn
	}
	return group, rows.Err()
}
