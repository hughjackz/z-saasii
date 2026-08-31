package model

import "time"

// ─── Role / User ────────────────────────────────────────────────────────────

type Role string

const (
	RoleCSAdmin Role = "CS_Admin"
	RoleCPOP    Role = "CP_OP"
	RoleCPOM    Role = "CP_OM"
)

// IsAdminLike returns true for roles that have admin-level access
func (r Role) IsAdminLike() bool { return r == RoleCSAdmin }

// IsOperatorLike returns true for roles that manage a tenant
func (r Role) IsOperatorLike() bool { return r == RoleCSAdmin || r == RoleCPOP }

// TenantScoped returns true if this role's data is scoped to a tenant
func (r Role) TenantScoped() bool { return r == RoleCPOP || r == RoleCPOM }

type User struct {
	ID           string    `db:"id"            json:"id"`
	Username     string    `db:"username"      json:"username"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Name         string    `db:"name"          json:"name"`
	CertGroup  string     `db:"cert_group"   json:"certGroup,omitempty"`
	Role         Role      `db:"role"          json:"role"`
	Company      string    `db:"company"       json:"company"`
	Email        string    `db:"email"         json:"email"`
	Contact      string    `db:"contact"       json:"contact"`
	Enabled      bool      `db:"enabled"       json:"enabled"`
	ParentID     *string   `db:"parent_id"     json:"parentId,omitempty"`  // CP_OM → CP_OP id
	TenantID     *string   `db:"tenant_id"     json:"tenantId,omitempty"`  // CP_OP=own id, CP_OM=parent's id, CS_Admin=NULL
	CreatedBy    *string   `db:"created_by"    json:"createdBy,omitempty"` // 创建者 id
	Permissions  string    `db:"permissions"   json:"-"`
	CreatedAt    time.Time `db:"created_at"    json:"createdAt"`
	UpdatedAt    time.Time `db:"updated_at"    json:"updatedAt"`
}

// UserResponse is the safe JSON payload (no password hash)
type UserResponse struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Name        string    `json:"name"`
	Role        Role      `json:"role"`
	Company     string    `json:"company"`
	Email       string    `json:"email"`
	Contact     string    `json:"contact"`
	Enabled     bool      `json:"enabled"`
	ParentID    *string   `json:"parentId,omitempty"`
	TenantID    *string   `json:"tenantId,omitempty"`
	CreatedBy   *string   `json:"createdBy,omitempty"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ─── Action ─────────────────────────────────────────────────────────────────

type Action struct {
	ID     int    `db:"id"      json:"id"`
	UserID string `db:"user_id" json:"userId"`
	Module string `db:"module"  json:"module"`
	Allow  bool   `db:"allow"   json:"allow"`
}

// ─── Device ──────────────────────────────────────────────────────────────────

type Device struct {
	ID                string     `db:"id"                 json:"id"`
	Name              string     `db:"name"               json:"name"`
	CertGroup  string     `db:"cert_group"   json:"certGroup,omitempty"`
	Protocol          string     `db:"protocol"           json:"protocol"`
	Location          string     `db:"location"           json:"location"`
	Enabled           bool       `db:"enabled"            json:"enabled"`
	HeartbeatInterval int        `db:"heartbeat_interval" json:"heartbeatInterval"`
	OwnerID           *string    `db:"owner_id"           json:"ownerId,omitempty"`
	OwnerName         string     `db:"owner_name"         json:"ownerName,omitempty"`
	TenantID          string     `db:"tenant_id"          json:"tenantId"`
	Status            string     `db:"status"             json:"status"`
	Online            bool       `db:"-"                  json:"online"`    // connection state from WS hub
	LastHeartbeat     *time.Time `db:"last_heartbeat"     json:"lastHeartbeat,omitempty"`
	CreatedAt         time.Time  `db:"created_at"         json:"createdAt"`
	UpdatedAt         time.Time  `db:"updated_at"         json:"updatedAt"`
}

// ─── Transaction ─────────────────────────────────────────────────────────────

type Transaction struct {
	ID            int64      `db:"id"              json:"id"`
	TransactionID int        `db:"transaction_id"  json:"transactionId"`
	ChargePointID string     `db:"charge_point_id" json:"chargePointId"`
	ConnectorID   int        `db:"connector_id"    json:"connectorId"`
	TenantID      string     `db:"tenant_id"       json:"tenantId"`
	IDTag         string     `db:"id_tag"          json:"idTag"`
	StartTime     time.Time  `db:"start_time"      json:"startTime"`
	StopTime      *time.Time `db:"stop_time"       json:"stopTime,omitempty"`
	StartMeter    float64    `db:"start_meter"     json:"startMeter"`
	StopMeter     *float64   `db:"stop_meter"      json:"stopMeter,omitempty"`
	StopReason    string     `db:"stop_reason"     json:"stopReason,omitempty"`
	Active        bool       `db:"active"          json:"active"`
}

// ─── Certificate ─────────────────────────────────────────────────────────────

type Certificate struct {
	ID                 string     `db:"id"                  json:"id"`
	Name               string     `db:"name"                json:"name"`
	CertGroup          string     `db:"cert_group"          json:"certGroup,omitempty"`
	Type               string     `db:"type"                json:"type"`
	Content            string     `db:"content"             json:"-"`
	PrivateKey         string     `db:"private_key"         json:"-"`
	KeyPassphrase  string     `db:"key_passphrase"      json:"-"`
	FilePath           string     `db:"file_path"           json:"filePath,omitempty"`
	PrivateKeyPath     string     `db:"private_key_path"    json:"privateKeyPath,omitempty"`
	SerialNumber       string     `db:"serial_number"       json:"serialNumber,omitempty"`
	IssuerName         string     `db:"issuer_name"         json:"issuerName,omitempty"`
	SubjectName        string     `db:"subject_name"        json:"subjectName,omitempty"`
	PublicKey          string     `db:"public_key"          json:"publicKey,omitempty"`
	SignatureAlgorithm string     `db:"signature_algorithm" json:"signatureAlgorithm,omitempty"`
	HashAlgorithm      string     `db:"hash_algorithm"      json:"hashAlgorithm,omitempty"`
	IssuerNameHash     string     `db:"issuer_name_hash"    json:"issuerNameHash,omitempty"`
	IssuerKeyHash      string     `db:"issuer_key_hash"     json:"issuerKeyHash,omitempty"`
	ValidFrom          *time.Time `db:"valid_from"          json:"validFrom,omitempty"`
	ValidTo            *time.Time `db:"valid_to"            json:"validTo,omitempty"`
	Enabled            bool       `db:"enabled"             json:"enabled"`
	UploadedAt         time.Time  `db:"uploaded_at"         json:"uploadedAt"`
	OwnerID            string     `db:"owner_id"            json:"ownerId"`
	OwnerName          string     `db:"owner_name"          json:"ownerName,omitempty"`
	TenantID           string     `db:"tenant_id"           json:"tenantId"`
}

// ─── IDTag

// ─── IDTag ───────────────────────────────────────────────────────────────────

type IDTag struct {
	ID          string     `db:"id"             json:"id"`
	TagID       string     `db:"tag_id"         json:"tagId"`
	ParentTagID *string    `db:"parent_tag_id"  json:"parentTagId,omitempty"`
	Status      string     `db:"status"         json:"status"`
	ExpiryTime  *time.Time `db:"expiry_time"    json:"expiryTime,omitempty"`
	OwnerID     string     `db:"owner_id"       json:"ownerId"`
	OwnerName   string     `db:"owner_name"     json:"ownerName,omitempty"`
	TenantID    string     `db:"tenant_id"      json:"tenantId"`
	CreatedAt   time.Time  `db:"created_at"     json:"createdAt"`
	UpdatedAt   time.Time  `db:"updated_at"     json:"updatedAt"`
}

// ─── ChargingProfile ─────────────────────────────────────────────────────────

type ChargingProfile struct {
	ID         string    `db:"id"          json:"id"`
	Name       string    `db:"name"         json:"name"`
	CertGroup  string     `db:"cert_group"   json:"certGroup,omitempty"`
	Purpose    string    `db:"purpose"      json:"purpose"`
	Content    string    `db:"content"      json:"-"`
	OwnerID    string    `db:"owner_id"     json:"ownerId"`
	OwnerName  string    `db:"owner_name"   json:"ownerName,omitempty"`
	TenantID   string    `db:"tenant_id"    json:"tenantId"`
	ImportedAt time.Time `db:"imported_at"  json:"importedAt"`
}

// ─── Event ───────────────────────────────────────────────────────────────────

type Event struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Device  string `json:"device"`
	Message string `json:"message"`
}

// ─── Auth ────────────────────────────────────────────────────────────────────

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// ─── VDV261 ──────────────────────────────────────────────────────────────────

type VDVProfile struct {
	ID          string    `db:"id"           json:"id"`
	Name        string    `db:"name"         json:"name"`
	DriveOff    string    `db:"driveoff"     json:"driveoff"`
	PrecDsrd    int       `db:"prec_dsrd"    json:"precDsrd"`
	PrecHvac    int       `db:"prec_hvac"    json:"precHvac"`
	AmbientTemp int       `db:"ambienttemp"   json:"ambientTemp"`
	TenantID    string    `db:"tenant_id"    json:"tenantId"`
	CPOPName    string    `db:"cpop_name"    json:"cpopName,omitempty"`
	CreatedAt   time.Time `db:"created_at"   json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at"   json:"updatedAt"`
}

type VDVCarInfo struct {
	ID             string    `db:"id"               json:"id"`
	VIN            string    `db:"vin"              json:"vin"`
	Password       string    `db:"password"         json:"password"`
	EVCCID         string    `db:"evccid"           json:"evccid,omitempty"`
	Odo            int       `db:"odo"              json:"odo"`
	VDVProfileID   *string   `db:"vdv_profile_id"   json:"vdvProfileId,omitempty"`
	VDVProfileName string    `db:"vdv_profile_name" json:"vdvProfileName,omitempty"`
	CPOPName       string    `db:"cpop_name"        json:"cpopName,omitempty"`
	TenantID       string    `db:"tenant_id"        json:"tenantId"`
	CreatedAt      time.Time `db:"created_at"       json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at"       json:"updatedAt"`
}

// ─── OCPP socket message ─────────────────────────────────────────────────────

type OCPPRequest struct {
	CPOPName   string      `json:"CP-OP_name"`
	CPUserName string      `json:"CP-User_name"`
	Action     string      `json:"Action"`
	Payload    interface{} `json:"payload"`
}

type OCPPResponse struct {
	Action  string      `json:"Action"`
	Status  string      `json:"status"`
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}
