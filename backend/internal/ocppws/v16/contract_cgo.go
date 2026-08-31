package v16

// contract_cgo.go — CGO wrapper for the C EXI static library (clib/exi_lib.a).
// Implements ContractGenerator using SA_EXI_Process for ISO 15118-2 EXI
// encoding/decoding of CertificateInstallationReq/Res messages.
//
// C++ bridge (exi_bridge.cpp) is compiled as C++ by CGO (g++ for .cpp),
// crossing the C/C++ boundary.  The Go code calls the cgo_SA_* wrappers.
//
// The C library expects:
//   - 13 DER certificates set via SA_Set_DER_Cert() before processing
//   - 3 PKCS#8 DER private keys set via SA_Set_PKCS8_PriKey() before processing
//   - SA_EXI_Process() called every ~10ms (blocking, needs dedicated polling)
//
// Ref: README 4.2.9.5, backend/clib/exi_lib.h

/*
#cgo LDFLAGS: ${SRCDIR}/../../../clib/exi_lib.a

// C-linkage bridge functions — implemented in exi_bridge.cpp (compiled as C++)
// because exi_lib.h uses C++ syntax (extern "C").
int  cgo_SA_Set_PKCS8_PriKey(int keyType, unsigned char *keyBuf, int keyLen);
int  cgo_SA_Set_DER_Cert(int certType, unsigned char *certBuf, int certLen);
int  cgo_SA_Start_Decode_ExiRequest(unsigned char *buf, int len);
int  cgo_SA_Get_Encode_ExiResponse(int *responseResult, unsigned char *buf, int *len);
void cgo_SA_EXI_Process();
*/
import "C"

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/repository"
)

// ─── C enum values (from exi_lib.h SA_CERT_TYPE / SA_PRIKEY_TYPE) ────────────
// Duplicated as Go constants because the C header uses C++ syntax and the
// bridge (exi_bridge.cpp) swallows the enum definitions.

// SA_CERT_TYPE
const (
	cCertMORoot             = 0
	cCertOEMRoot            = 1
	cCertOEMSubCA1          = 2
	cCertOEMSubCA2          = 3
	cCertContractSubCA1Inst = 4
	cCertContractSubCA2Inst = 5
	cCertContractLeafInst   = 6
	cCertContractSubCA1Upd  = 7
	cCertContractSubCA2Upd  = 8
	cCertContractLeafUpd    = 9
	cCertCPSSubCA1          = 10
	cCertCPSSubCA2          = 11
	cCertCPSLeaf            = 12
)

// SA_PRIKEY_TYPE
const (
	cPriKeyCPSLeaf          = 0
	cPriKeyContractLeafInst = 1
	cPriKeyContractLeafUpd  = 2
)

// ─── DB type → C enum mapping ─────────────────────────────────────────────────

var dbToCCertType = map[string]int{
	// MO group
	"MO-root-cert": cCertMORoot,
	"MO-sub1-cert": cCertContractSubCA1Upd,
	"MO-sub2-cert": cCertContractSubCA2Upd,

	// V2G/OEM group
	"V2G-root-cert": cCertOEMRoot,

	// OEM group
	"OEM-root-cert": cCertOEMRoot,
	"OEM-sub1-cert": cCertOEMSubCA1,
	"OEM-sub2-cert": cCertOEMSubCA2,

	// CPO group → Contract Install chain
	"CPO-sub1-cert":      cCertContractSubCA1Inst,
	"CPO-sub2-cert":      cCertContractSubCA2Inst,
	"Contract-leaf-cert": cCertContractLeafInst,

	// CPS group
	"CPS-sub1-cert":  cCertCPSSubCA1,
	"CPS-sub2-cert":  cCertCPSSubCA2,
	"CPS-leaf-cert":  cCertCPSLeaf,
	"SECC-leaf-cert": cCertCPSLeaf,

	// Legacy mappings (for backward compatibility with old DB records)
	"MO Root":   cCertMORoot,
	"V2G Root":  cCertOEMRoot,
	"V2G Sub1":  cCertOEMSubCA1,
	"V2G Sub2":  cCertOEMSubCA2,
	"CPS Sub1":  cCertCPSSubCA1,
	"CPS Sub2":  cCertCPSSubCA2,
	"CPS Root":  cCertMORoot,
	"SECC Leaf": cCertCPSLeaf,
}

// dbToCKeyTypes maps a DB cert type to one or more C SA_PRIKEY_TYPE values.
// The C library requires all 3 private keys to be set (exi_lib.h):
//
//	PriKey_CPS_Leaf              (0) — CPS leaf private key
//	PriKey_Contarct_Leaf_Install (1) — Contract leaf private key
//	PriKey_Contarct_Leaf_Update  (2) — Contract leaf private key (same key as install)
//
// A single DB cert may supply the key for multiple C key slots
// (e.g. Contract-leaf-cert feeds both Install and Update).
var dbToCKeyTypes = map[string][]int{
	// CPO Sub2 private key → CPS Leaf signing key
	//"CPO-sub2-cert": {cPriKeyCPSLeaf},
	// CPS Leaf private key → also maps to CPS Leaf signing key
	"CPS-leaf-cert": {cPriKeyCPSLeaf},
	// Contract Leaf private key → both Install and Update signing keys (same key)
	"Contract-leaf-cert": {cPriKeyContractLeafInst, cPriKeyContractLeafUpd},
	// Legacy mappings
	//"CPS Sub2": {cPriKeyCPSLeaf},
}

// ─── CGOContractGenerator ─────────────────────────────────────────────────────

// CGOContractGenerator implements ContractGenerator by calling the C EXI
// static library via the cgo_SA_* bridge. Thread-safe (mutex-guarded).
type CGOContractGenerator struct {
	mu       sync.Mutex
	initDone bool
}

// Initialize loads all certificates and private keys for the given tenant
// and configures the C library. Must be called before Generate().
// Safe to call multiple times; subsequent calls are no-ops.
func (g *CGOContractGenerator) Initialize(tenantID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.initDone {
		return nil
	}

	certs, err := repository.ListCertificates(model.RoleCSAdmin, "", tenantID, "")
	if err != nil {
		return fmt.Errorf("contract_cgo: list certs: %w", err)
	}

	certSet, certSkipped := 0, 0
	for _, cert := range certs {
		cType, ok := dbToCCertType[cert.Type]
		if !ok {
			certSkipped++
			log.Printf("[contract_cgo] cert %q type=%q — no C enum mapping, skipping",
				cert.Name, cert.Type)
			continue
		}

		der, err := pemCertToDER(cert.Content)
		if err != nil {
			log.Printf("[contract_cgo] cert %q PEM→DER: %v", cert.Name, err)
			continue
		}

		ret := C.cgo_SA_Set_DER_Cert(C.int(cType),
			(*C.uchar)(unsafe.Pointer(&der[0])), C.int(len(der)))
		if ret != 0 {
			log.Printf("[contract_cgo] cgo_SA_Set_DER_Cert(%q → %d) failed: ret=%d",
				cert.Name, cType, int(ret))
		} else {
			certSet++
			log.Printf("[contract_cgo] cgo_SA_Set_DER_Cert(%q → %d) OK (%d bytes)",
				cert.Name, cType, len(der))
		}
	}

	// Set private keys.
	// The C library requires all 3 SA_PRIKEY_TYPE slots to be filled
	// (PriKey_CPS_Leaf, PriKey_Contarct_Leaf_Install, PriKey_Contarct_Leaf_Update).
	// One DB cert may supply the key for multiple C slots — e.g.
	// "Contract-leaf-cert" feeds both Install and Update with the same key.
	keySet := 0
	for _, cert := range certs {
		kTypes, ok := dbToCKeyTypes[cert.Type]
		if !ok || cert.PrivateKey == "" {
			continue
		}

		keyPEM := cert.PrivateKey
		if cert.KeyPassphrase != "" {
			decrypted, err := decryptPEMString(keyPEM, cert.KeyPassphrase)
			if err != nil {
				log.Printf("[contract_cgo] decrypt key for %q: %v", cert.Name, err)
				continue
			}
			keyPEM = decrypted
		}

		der, err := pemKeyToPKCS8DER(keyPEM)
		if err != nil {
			log.Printf("[contract_cgo] key %q PEM→PKCS#8 DER: %v", cert.Name, err)
			continue
		}

		for _, kType := range kTypes {
			ret := C.cgo_SA_Set_PKCS8_PriKey(C.int(kType),
				(*C.uchar)(unsafe.Pointer(&der[0])), C.int(len(der)))
			if ret != 0 {
				log.Printf("[contract_cgo] cgo_SA_Set_PKCS8_PriKey(%q → %d) failed: ret=%d",
					cert.Name, kType, int(ret))
			} else {
				keySet++
				log.Printf("[contract_cgo] cgo_SA_Set_PKCS8_PriKey(%q → %d) OK (%d bytes)",
					cert.Name, kType, len(der))
			}
		}
	}

	log.Printf("[contract_cgo] initialized: %d certs set, %d skipped, %d keys (tenant=%s)",
		certSet, certSkipped, keySet, tenantID)

	g.initDone = false
	return nil
}

// Generate implements ContractGenerator by calling the C EXI library.
//
// Flow:
//  1. Base64-decode exiRequest → binary
//  2. cgo_SA_Start_Decode_ExiRequest(binary)
//  3. Poll cgo_SA_EXI_Process() + cgo_SA_Get_Encode_ExiResponse() every ~10ms
//  4. When responseResult==1, Base64-encode the binary response → return
func (g *CGOContractGenerator) Generate(exiRequest string, _ []*model.Certificate) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	exiBin, err := base64.StdEncoding.DecodeString(exiRequest)
	if err != nil {
		return "", fmt.Errorf("exiRequest base64 decode: %w", err)
	}
	if len(exiBin) == 0 {
		return "", fmt.Errorf("exiRequest is empty after base64 decode")
	}

	log.Printf("[contract_cgo] Generate: base64Len=%d binLen=%d", len(exiRequest), len(exiBin))

	ret := C.cgo_SA_Start_Decode_ExiRequest(
		(*C.uchar)(unsafe.Pointer(&exiBin[0])), C.int(len(exiBin)))
	if ret != 0 {
		return "", fmt.Errorf("cgo_SA_Start_Decode_ExiRequest failed: ret=%d", int(ret))
	}

	// Poll for completion (10ms interval, 30s timeout)
	var responseResult C.int
	responseBuf := make([]byte, 5600) // max per Get15118EVCertificateResponse schema
	var responseLen C.int

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		C.cgo_SA_EXI_Process()

		r := C.cgo_SA_Get_Encode_ExiResponse(
			&responseResult,
			(*C.uchar)(unsafe.Pointer(&responseBuf[0])),
			&responseLen,
		)
		if r != 0 {
			return "", fmt.Errorf("cgo_SA_Get_Encode_ExiResponse failed: ret=%d", int(r))
		}

		if int(responseResult) == 1 {
			exiResponse := base64.StdEncoding.EncodeToString(responseBuf[:int(responseLen)])
			log.Printf("[contract_cgo] Generate: success, base64 respLen=%d", len(exiResponse))
			return exiResponse, nil
		}

		time.Sleep(10 * time.Millisecond)
	}

	return "", fmt.Errorf("EXI processing timed out after 30s")
}

// Reset clears the initialization flag so Initialize can be called again
// (e.g., after uploading new certificates).
func (g *CGOContractGenerator) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.initDone = false
	log.Printf("[contract_cgo] reset — re-initialize on next Generate")
}

// ─── PEM → DER helpers ────────────────────────────────────────────────────────

func pemCertToDER(pemData string) ([]byte, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}
	return block.Bytes, nil
}

func pemKeyToPKCS8DER(pemData string) ([]byte, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}

	// Already PKCS#8?
	if _, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return block.Bytes, nil
	}
	// EC private key → PKCS#8
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return x509.MarshalPKCS8PrivateKey(key)
	}
	// RSA private key → PKCS#8
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return x509.MarshalPKCS8PrivateKey(key)
	}
	return nil, fmt.Errorf("unsupported private key format (not PKCS#8, EC, or RSA)")
}

// decryptPEMString decrypts an encrypted PEM using the legacy OpenSSL
// EVP_BytesToKey (Proc-Type: 4,ENCRYPTED) format. Delegates to decryptPEM
// from secc.go.
func decryptPEMString(pemData, passphrase string) (string, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM")
	}
	if !strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED") {
		return pemData, nil
	}
	decrypted, err := decryptPEM(block, []byte(passphrase))
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  block.Type,
		Bytes: decrypted,
	})), nil
}

// Ensure CGOContractGenerator implements ContractGenerator.
var _ ContractGenerator = (*CGOContractGenerator)(nil)
