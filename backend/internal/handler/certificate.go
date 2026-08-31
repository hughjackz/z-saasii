package handler

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/repository"
)

func ListCertificates(c *gin.Context) {
	callerRole, callerID, tenantID := tenantDB(c)
	certType := c.Query("type")
	certs, err := repository.ListCertificates(callerRole, callerID, tenantID, certType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, certs)
}

func UploadCertificate(c *gin.Context) {
	_, _, tenantID := tenantDB(c)

	// Read certificate file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()
	content, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	certPEM := string(content)

	// Read private key file
	var privKeyPEM string
	pkFile, err := c.FormFile("private_key")
	if err == nil && pkFile != nil {
		pkF, _ := pkFile.Open()
		if pkF != nil {
			defer pkF.Close()
			pkBytes, _ := io.ReadAll(pkF)
			privKeyPEM = string(pkBytes)
		}
	}

	// Parse X.509 certificate (don't validate — just extract parameters)
	var serialNo, issuer, subject, pubKey, sigAlg string
	var hashAlg, issuerNameHash, issuerKeyHash string
	var validFrom, validTo *time.Time
	if block, _ := pem.Decode(content); block != nil {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			serialNo = cert.SerialNumber.String()
			issuer = cert.Issuer.String()
			subject = cert.Subject.String()
			sigAlg = cert.SignatureAlgorithm.String()
			hashAlg = "SHA256"
			// issuerNameHash: SHA-256 of DER-encoded issuer name
			ih := sha256.Sum256(cert.RawIssuer)
			issuerNameHash = hex.EncodeToString(ih[:])
			// issuerKeyHash: SHA-256 of BITSTRING value (excluding tag & length)
			// per OCPP 1.6 Security Whitepaper 6.1 / RFC 6960
			keyBytes := extractSPKIBitString(cert.RawSubjectPublicKeyInfo)
			kh := sha256.Sum256(keyBytes)
			issuerKeyHash = hex.EncodeToString(kh[:])
			if pub, e := x509.MarshalPKIXPublicKey(cert.PublicKey); e == nil {
				pubKey = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub}))
			}
			vf := cert.NotBefore.UTC()
			validFrom = &vf
			vt := cert.NotAfter.UTC()
			validTo = &vt
		}
	}

	name := c.PostForm("name")
	if name == "" {
		name = file.Filename
	}
	certGroup := c.PostForm("cert_group")
	certType := c.PostForm("type")

	// Per README 2.3.4.2: CPO-sub2-cert, CPS-leaf-cert, Contract-leaf-cert
	// MUST include a private key file.
	requireKey := certType == "CPO-sub2-cert" || certType == "CPS-leaf-cert" || certType == "Contract-leaf-cert"
	if requireKey && privKeyPEM == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "private_key file required for " + certType})
		return
	}

	// Determine tenant ID
	tid := tenantID
	if fv := c.PostForm("tenant_id"); fv != "" {
		tid = fv
	}

	cert := &model.Certificate{
		Name:           name,
		CertGroup:      certGroup,
		Type:           certType,
		Content:        certPEM,
		PrivateKey:     privKeyPEM,
		KeyPassphrase:  c.PostForm("key_passphrase"),
		SerialNumber:       serialNo,
		IssuerName:         issuer,
		SubjectName:        subject,
		PublicKey:          pubKey,
		SignatureAlgorithm: sigAlg,
		HashAlgorithm:      hashAlg,
		IssuerNameHash:     issuerNameHash,
		IssuerKeyHash:      issuerKeyHash,
		ValidFrom:          validFrom,
		ValidTo:            validTo,
		Enabled:            true,
		OwnerID:            tid,
		TenantID:           tid,
	}

	if err := repository.CreateCertificate(cert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cert)
}

func RenameCertificate(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	if err := repository.UpdateCertificateName(c.Param("id"), req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "renamed"})
}

// extractSPKIBitString extracts the raw key bytes from a DER-encoded
// SubjectPublicKeyInfo, stripping the SEQUENCE wrapper and BITSTRING tag/length.
// Used to compute issuerKeyHash per OCPP 1.6 Security Whitepaper §6.1 / RFC 6960.
func extractSPKIBitString(spki []byte) []byte {
	var inner struct {
		Algorithm asn1.RawValue
		PublicKey asn1.BitString
	}
	if _, err := asn1.Unmarshal(spki, &inner); err != nil {
		return spki
	}
	return inner.PublicKey.Bytes
}

// GET /api/certificates/:id/content
func GetCertificateContent(c *gin.Context) {
	id := c.Param("id")
	content, privKey, err := repository.GetCertificateContent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": content, "privateKey": privKey})
}

func DeleteCertificate(c *gin.Context) {
	id := c.Param("id")

	// Certificates and keys are stored in DB only — no local files to clean up.
	if err := repository.DeleteCertificate(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
