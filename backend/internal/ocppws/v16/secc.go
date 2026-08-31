package v16

// secc.go — SECC Leaf certificate signing (README 4.2.9.1).
//
// Three-step flow:
//   1. Frontend selects V2G Root/Sub1/Sub2 → records pending session → TriggerMessage → device
//   2. Device sends SignCertificate.req with CSR → SAAS signs with V2G Sub2 → stores SECC Leaf
//   3. SAAS sends CertificateSigned.req → device confirms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/repository"
)

// seccPending stores the V2G cert chain selected by the operator for SECC Leaf signing.
type seccPending struct {
	V2GRoot  string // cert name (from certificate library)
	V2GSub1  string
	V2GSub2  string
	CPOPName string
}

var (
	seccSessions   = make(map[string]*seccPending) // deviceID → pending session
	seccSessionsMu sync.Mutex
)

// RecordSECCSession stores the operator's V2G cert selections for a device.
func RecordSECCSession(deviceID, v2gRoot, v2gSub1, v2gSub2, cpopName string) {
	seccSessionsMu.Lock()
	defer seccSessionsMu.Unlock()
	seccSessions[deviceID] = &seccPending{
		V2GRoot: v2gRoot, V2GSub1: v2gSub1, V2GSub2: v2gSub2, CPOPName: cpopName,
	}
}

// GetSECCSession returns the pending session for a device (nil if none).
func GetSECCSession(deviceID string) *seccPending {
	seccSessionsMu.Lock()
	defer seccSessionsMu.Unlock()
	return seccSessions[deviceID]
}

// SignCSR signs the given PEM-encoded CSR using the V2G Sub2 private key
// and returns the signed certificate in PEM format.
// The signed cert is saved to resource/{cpopName}/certificate/{deviceName}_SECCLeaf_{serial}.pem
// and recorded in the database and cert_serial table.
func SignCSR(csrPEM, deviceName, tenantID string, sp *seccPending) (string, error) {
	// 1. Parse CSR
	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil {
		return "", fmt.Errorf("failed to decode CSR PEM")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse CSR: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return "", fmt.Errorf("CSR signature invalid: %w", err)
	}

	// 2. Get V2G Sub2 issuer certificate and private key
	// Look up by name "V2G_{name}_sub2" pattern or search cert library
	issuerCertPEM, issuerKeyPEM, keyPass, err := findCertAndKey(sp.V2GSub2)
	fmt.Println("V2G Sub2.filename:" + sp.V2GSub2)
	fmt.Println("V2G Sub2.certcontent:" + issuerCertPEM)
	fmt.Println("V2G Sub2.keycontent:" + issuerKeyPEM)

	if err != nil {
		return "", fmt.Errorf("V2G Sub2 not found: %w", err)
	}

	issuerBlock, _ := pem.Decode([]byte(issuerCertPEM))
	if issuerBlock == nil {
		return "", fmt.Errorf("failed to decode issuer cert")
	}
	issuerCert, err := x509.ParseCertificate(issuerBlock.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse issuer cert: %w", err)
	}

	// Parse private key — find the actual key block (may be preceded by EC PARAMETERS)
	var issuerKey interface{}
	rest := []byte(issuerKeyPEM)
	var keyErr error
	for len(rest) > 0 {
		keyBlock, remaining := pem.Decode(rest)
		if keyBlock == nil {
			break
		}
		rest = remaining

		// Skip non-key blocks (EC PARAMETERS etc.)
		if !strings.Contains(keyBlock.Type, "PRIVATE KEY") && !strings.Contains(keyBlock.Type, "RSA KEY") {
			continue
		}

		keyBytes := keyBlock.Bytes
		if strings.Contains(keyBlock.Headers["Proc-Type"], "ENCRYPTED") {
			var decErr error
			keyBytes, decErr = decryptPEM(keyBlock, []byte(keyPass))
			if decErr != nil {
				return "", fmt.Errorf("private key decryption failed: %w", decErr)
			}
		}

		keyErr = nil
		switch keyBlock.Type {
		case "EC PRIVATE KEY":
			issuerKey, keyErr = x509.ParseECPrivateKey(keyBytes)
		case "RSA PRIVATE KEY":
			issuerKey, keyErr = x509.ParsePKCS1PrivateKey(keyBytes)
		default: // "PRIVATE KEY" (PKCS8)
			issuerKey, keyErr = x509.ParsePKCS8PrivateKey(keyBytes)
			if keyErr != nil {
				issuerKey, keyErr = x509.ParseECPrivateKey(keyBytes)
				if keyErr != nil {
					issuerKey, keyErr = x509.ParsePKCS1PrivateKey(keyBytes)
				}
			}
		}
		if keyErr == nil {
			break // successfully parsed
		}
	}
	if issuerKey == nil || keyErr != nil {
		return "", fmt.Errorf("failed to parse private key: %w", keyErr)
	}

	// 3. Get next serial number
	serialNo, err := repository.GetNextSerialNumber(tenantID, "SECCLeaf")
	if err != nil {
		return "", fmt.Errorf("serial number: %w", err)
	}

	// 4. Create certificate template from CSR
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(serialNo),
		Subject:               csr.Subject,
		PublicKey:             csr.PublicKey,
		PublicKeyAlgorithm:    csr.PublicKeyAlgorithm,
		SignatureAlgorithm:    issuerCert.SignatureAlgorithm,
		NotBefore:             issuerCert.NotBefore,
		NotAfter:              issuerCert.NotAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// 5. Sign certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, issuerCert, csr.PublicKey, issuerKey)
	if err != nil {
		return "", fmt.Errorf("certificate signing failed: %w", err)
	}

	signedPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))

	fileName := fmt.Sprintf("%s_SECCLeaf_%d.pem", deviceName, serialNo)

	// 6. Store in database only (no local file — README 2.3.4.2)
	notBefore := issuerCert.NotBefore.UTC()
	notAfter := issuerCert.NotAfter.UTC()
	cert := &model.Certificate{
		Name:           fileName,
		CertGroup:      deviceName,
		Type:           "SECC-leaf-cert",
		Content:        signedPEM,
		PrivateKey:     "", // SECC Leaf private key stays on device
		SerialNumber:       fmt.Sprintf("%x", serialNo),
		IssuerName:         issuerCert.Issuer.String(),
		SubjectName:        csr.Subject.String(),
		PublicKey:          string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: csr.RawSubjectPublicKeyInfo})),
		SignatureAlgorithm: issuerCert.SignatureAlgorithm.String(),
		ValidFrom:          &notBefore,
		ValidTo:            &notAfter,
		Enabled:            true,
		OwnerID:            tenantID,
		TenantID:           tenantID,
	}
	if err := repository.CreateCertificate(cert); err != nil {
		return "", fmt.Errorf("save to DB: %w", err)
	}

	return signedPEM, nil
}

// decryptPEM decrypts an encrypted PEM block using the legacy OpenSSL format
// (Proc-Type: 4,ENCRYPTED / DEK-Info: AES-128-CBC,<iv>).
func decryptPEM(block *pem.Block, password []byte) ([]byte, error) {
	if !strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED") {
		return block.Bytes, nil // not encrypted
	}
	dek := block.Headers["DEK-Info"]
	if dek == "" {
		return nil, fmt.Errorf("missing DEK-Info header")
	}
	parts := strings.SplitN(dek, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid DEK-Info")
	}
	iv, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid IV: %w", err)
	}

	// Key derivation (legacy OpenSSL EVP_BytesToKey with MD5)
	hash := md5.New
	keyLen := 16 // AES-128
	salt := iv[:8]
	derived := make([]byte, 0, keyLen+len(iv))
	var dgst []byte
	for len(derived) < keyLen+len(iv) {
		h := hash()
		if len(dgst) > 0 {
			h.Write(dgst)
		}
		h.Write(password)
		h.Write(salt)
		dgst = h.Sum(nil)
		derived = append(derived, dgst...)
	}
	key := derived[:keyLen]

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	cbc := cipher.NewCBCDecrypter(blockCipher, iv)
	decrypted := make([]byte, len(block.Bytes))
	cbc.CryptBlocks(decrypted, block.Bytes)

	// Remove PKCS#7 padding
	if len(decrypted) == 0 {
		return nil, fmt.Errorf("empty decrypted data")
	}
	padLen := int(decrypted[len(decrypted)-1])
	if padLen > len(decrypted) || padLen > aes.BlockSize {
		return nil, fmt.Errorf("invalid padding")
	}
	return decrypted[:len(decrypted)-padLen], nil
}

// findCertAndKey looks up a certificate, private key, and passphrase from the database by name.
func findCertAndKey(name string) (certPEM, keyPEM, passphrase string, err error) {
	certs, err := repository.ListCertificates("CS_Admin", "", "", "")
	if err != nil {
		return "", "", "", err
	}
	for _, c := range certs {
		if c.Name == name {
			content, priv, pass, e := repository.GetCertKeyAndPassphrase(c.ID)
			return content, priv, pass, e
		}
	}
	return "", "", "", fmt.Errorf("certificate %q not found", name)
}
