-- db-saas-ocpp-dev  (InnoDB, utf8mb4)
CREATE DATABASE IF NOT EXISTS `db-saas-ocpp-dev`
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `db-saas-ocpp-dev`;

-- ─── 5.1 role (users) ───────────────────────────────────────────────────────
-- role hierarchy: CS_Admin → CP_OP → CP_OM
--   CS_Admin: tenant_id IS NULL, manages CP_OP users only
--   CP_OP:    tenant_id = own id, manages CP_OM users under them
--   CP_OM:    tenant_id = parent CP_OP id, no user management
CREATE TABLE IF NOT EXISTS `role` (
  `id`            VARCHAR(36)  NOT NULL,
  `username`      VARCHAR(64)  NOT NULL UNIQUE,
  `password_hash` VARCHAR(255) NOT NULL,
  `name`          VARCHAR(128) NOT NULL DEFAULT '',
  `role`          ENUM('CS_Admin','CP_OP','CP_OM') NOT NULL DEFAULT 'CP_OM',
  `company`       VARCHAR(128) NOT NULL DEFAULT '',
  `email`         VARCHAR(128) NOT NULL DEFAULT '',
  `contact`       VARCHAR(64)  NOT NULL DEFAULT '',
  `enabled`       TINYINT(1)   NOT NULL DEFAULT 1,
  `parent_id`     VARCHAR(36)  NULL,           -- CP_OM → CP_OP
  `tenant_id`     VARCHAR(36)  NULL,           -- CP_OP=own id, CP_OM=parent.id, CS_Admin=NULL
  `created_by`    VARCHAR(36)  NULL,
  `permissions`   JSON         NOT NULL DEFAULT ('[]'),
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX idx_role_parent (`parent_id`),
  INDEX idx_role_tenant (`tenant_id`),
  CONSTRAINT fk_role_parent FOREIGN KEY (`parent_id`) REFERENCES `role`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── 5.2 action (per-user module permissions — denormalised in JSON on role, ───
--         but kept here as a log / audit table if needed)
CREATE TABLE IF NOT EXISTS `action` (
  `id`      INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `user_id` VARCHAR(36)  NOT NULL,
  `module`  VARCHAR(64)  NOT NULL,   -- e.g. ocpp.configuration
  `allow`   TINYINT(1)   NOT NULL DEFAULT 1,
  INDEX idx_action_user (`user_id`),
  CONSTRAINT fk_action_user FOREIGN KEY (`user_id`) REFERENCES `role`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── 5.4 device ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS `device` (
  `id`                 VARCHAR(36)  NOT NULL,
  `name`               VARCHAR(64)  NOT NULL,
  `protocol`           VARCHAR(16)  NOT NULL DEFAULT 'OCPP16',
  `location`           VARCHAR(128) NOT NULL DEFAULT '',
  `enabled`            TINYINT(1)   NOT NULL DEFAULT 1,   -- 0=rejected, 1=accepted
  `heartbeat_interval` INT          NOT NULL DEFAULT 60,
  `owner_id`           VARCHAR(36)  NULL,                  -- CP_OM who owns this device
  `tenant_id`          VARCHAR(36)  NOT NULL,              -- CP_OP's id (data isolation)
  `status`             VARCHAR(32)  NOT NULL DEFAULT 'Unavailable',
  `last_heartbeat`     DATETIME     NULL,
  `created_at`         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY uq_device_name_tenant (`name`, `tenant_id`),
  INDEX idx_device_owner (`owner_id`),
  INDEX idx_device_tenant (`tenant_id`),
  CONSTRAINT fk_device_owner FOREIGN KEY (`owner_id`) REFERENCES `role`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── 5.3 transaction ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS `transaction` (
  `id`              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `transaction_id`  INT          NOT NULL,
  `charge_point_id` VARCHAR(64)  NOT NULL,
  `connector_id`    INT          NOT NULL DEFAULT 1,
  `tenant_id`       VARCHAR(36)  NOT NULL,                  -- CP_OP's id
  `id_tag`          VARCHAR(64)  NOT NULL DEFAULT '',
  `start_time`      DATETIME     NOT NULL,
  `stop_time`       DATETIME     NULL,
  `start_meter`     DOUBLE       NOT NULL DEFAULT 0,
  `stop_meter`      DOUBLE       NULL,
  `stop_reason`     VARCHAR(64)  NOT NULL DEFAULT '',
  `active`          TINYINT(1)   NOT NULL DEFAULT 1,
  UNIQUE KEY uq_tx (`transaction_id`, `charge_point_id`, `tenant_id`),
  INDEX idx_tx_cp     (`charge_point_id`),
  INDEX idx_tx_active  (`active`),
  INDEX idx_tx_tenant  (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── meter_value (MeterValues storage) ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS `meter_value` (
  `id`             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `tenant_id`      VARCHAR(36)  NOT NULL,
  `transaction_id` INT          NULL,
  `connector_id`   INT          NOT NULL DEFAULT 1,
  `value`          JSON         NOT NULL,     -- sampledValue array from OCPP
  `created_at`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_mv_tenant (`tenant_id`),
  INDEX idx_mv_tx     (`transaction_id`),
  INDEX idx_mv_created (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── 5.5 certificate ─────────────────────────────────────────────────────────
-- Certificates and keys are stored in DB only (content / private_key columns).
-- Local file storage is no longer used — file_path and private_key_path are
-- kept for backward compatibility with existing records (default '').
CREATE TABLE IF NOT EXISTS `certificate` (
  `id`                  VARCHAR(36)   NOT NULL,
  `name`                VARCHAR(128)  NOT NULL,
  `cert_group`          VARCHAR(128)  NOT NULL DEFAULT '',
  `type`                VARCHAR(64)   NOT NULL,            -- e.g. V2G-root-cert, CPO-sub2-cert...
  `content`             MEDIUMTEXT    NOT NULL,
  `private_key`         MEDIUMTEXT    NOT NULL,
  `key_passphrase`      VARCHAR(128)  NOT NULL DEFAULT '',
  `file_path`           VARCHAR(512)  NOT NULL DEFAULT '', -- deprecated: DB-only since refactor
  `private_key_path`    VARCHAR(512)  NOT NULL DEFAULT '', -- deprecated: DB-only since refactor
  `serial_number`       VARCHAR(64)   NOT NULL DEFAULT '',
  `issuer_name`         VARCHAR(256)  NOT NULL DEFAULT '',
  `subject_name`        VARCHAR(256)  NOT NULL DEFAULT '',
  `public_key`          TEXT          NOT NULL,
  `signature_algorithm` VARCHAR(64)   NOT NULL DEFAULT '',
  `hash_algorithm`      VARCHAR(16)   NOT NULL DEFAULT 'SHA256',
  `issuer_name_hash`    VARCHAR(128)  NOT NULL DEFAULT '',
  `issuer_key_hash`     VARCHAR(128)  NOT NULL DEFAULT '',
  `valid_from`          DATETIME      NULL,
  `valid_to`            DATETIME      NULL,
  `enabled`             TINYINT(1)    NOT NULL DEFAULT 1,
  `uploaded_at`         DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `owner_id`            VARCHAR(36)   NOT NULL,
  `tenant_id`           VARCHAR(36)   NOT NULL,
  PRIMARY KEY (`id`),
  INDEX idx_cert_type   (`type`),
  INDEX idx_cert_owner  (`owner_id`),
  INDEX idx_cert_tenant (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── idtag ───────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS `idtag` (
  `id`            VARCHAR(36)  NOT NULL,
  `tag_id`        VARCHAR(64)  NOT NULL,
  `parent_tag_id` VARCHAR(64)  NULL,
  `status`        ENUM('Valid','Blocked','Expired') NOT NULL DEFAULT 'Valid',
  `expiry_time`   DATETIME     NULL,
  `owner_id`      VARCHAR(36)  NOT NULL,
  `tenant_id`     VARCHAR(36)  NOT NULL,             -- CP_OP's id
  `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY uq_idtag_tenant (`tag_id`, `tenant_id`),
  INDEX idx_idtag_owner (`owner_id`),
  INDEX idx_idtag_tenant (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── charging_profile ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS `charging_profile` (
  `id`          VARCHAR(36)   NOT NULL,
  `name`        VARCHAR(128)  NOT NULL,
  `purpose`     VARCHAR(64)   NOT NULL DEFAULT '',
  `content`     MEDIUMTEXT    NOT NULL,   -- JSON of OCPP ChargingProfile
  `owner_id`    VARCHAR(36)   NOT NULL,
  `tenant_id`   VARCHAR(36)   NOT NULL,   -- CP_OP's id
  `imported_at` DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX idx_profile_owner (`owner_id`),
  INDEX idx_profile_tenant (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── 5.7 VDVProfile ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS `vdv_profile` (
  `id`           VARCHAR(36)  NOT NULL,
  `name`         VARCHAR(128) NOT NULL,
  `driveoff`     VARCHAR(5)   NOT NULL DEFAULT '00:00',
  `prec_dsrd`    INT          NOT NULL DEFAULT 0,
  `prec_hvac`    INT          NOT NULL DEFAULT 0,
  `ambienttemp`  INT          NOT NULL DEFAULT 22,
  `tenant_id`    VARCHAR(36)  NOT NULL,
  `created_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX idx_vdvp_tenant (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── 5.8 VDVCarInfo ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS `vdv_carinfo` (
  `id`             VARCHAR(36)  NOT NULL,
  `vin`            VARCHAR(64)  NOT NULL UNIQUE,
  `password`       VARCHAR(128) NOT NULL,
  `evccid`         VARCHAR(128) NOT NULL DEFAULT '',
  `odo`            INT          NOT NULL DEFAULT 0,
  `vdv_profile_id` VARCHAR(36)  NULL,
  `tenant_id`      VARCHAR(36)  NOT NULL,
  `created_at`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX idx_vdvc_tenant (`tenant_id`),
  INDEX idx_vdvc_vin    (`vin`),
  CONSTRAINT fk_vdvc_profile FOREIGN KEY (`vdv_profile_id`) REFERENCES `vdv_profile`(`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── event_log (event persistence for EVENT view) ─────────────────────────
CREATE TABLE IF NOT EXISTS `event_log` (
  `id`         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `tenant_id`  VARCHAR(36)  NOT NULL DEFAULT '',
  `time`       DATETIME     NOT NULL,
  `level`      VARCHAR(16)  NOT NULL DEFAULT 'info',
  `device`     VARCHAR(128) NOT NULL DEFAULT '',
  `message`    TEXT         NOT NULL,
  INDEX idx_el_tenant (`tenant_id`),
  INDEX idx_el_time   (`time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── contract_cert_group (2.3.2.4.e / 4.2.9.5) ───────────────────────────
-- Stores user-selected certificate group per device for ContractGenerate.
-- When the device sends Get15118EVCertificate, the stored group is used
-- to build the contract certificate response.
CREATE TABLE IF NOT EXISTS `contract_cert_group` (
  `id`          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `device_id`   VARCHAR(36)  NOT NULL,
  `tenant_id`   VARCHAR(36)  NOT NULL,
  `cert_type`   VARCHAR(64)  NOT NULL,   -- e.g. "V2G-root-cert"
  `cert_name`   VARCHAR(128) NOT NULL,   -- the selected certificate name
  `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_ccg_device_type (`device_id`, `cert_type`),
  INDEX idx_ccg_tenant (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── cert_serial (SECC Leaf certificate serial number tracking) ────────────
-- serial counter for SECC Leaf certs; initial value 0x13155BC = 20012476
CREATE TABLE IF NOT EXISTS `cert_serial` (
  `id`          INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  `tenant_id`   VARCHAR(36)  NOT NULL,
  `cert_type`   VARCHAR(64)  NOT NULL DEFAULT 'SECCLeaf',
  `serial_no`   BIGINT       NOT NULL DEFAULT 20012476,
  `updated_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_cs_tenant (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─── Default CS_Admin user (password: Admin@1234) ───────────────────────────────
-- bcrypt hash of "Admin@1234"
INSERT IGNORE INTO `role`
  (id, username, password_hash, name, role, company, email, enabled, parent_id, tenant_id, created_by, permissions)
VALUES (
  'admin-0000-0000-0000-000000000001',
  'admin',
  '$2a$10$hYHUIOapENDge3V23uL/kOE2Tft2WvvvucEjjowG7WAwYBhwJsWDK',
  'System Administrator',
  'CS_Admin',
  'CSMS Corp',
  'admin@csms.local',
  1,
  NULL,
  NULL,
  'admin-0000-0000-0000-000000000001',
  '["ocpp.configuration","ocpp.transaction","ocpp.action","ocpp.maintenance","ocpp.pnc","ocpp.smartcharging","vdv261","management.users","management.devices","management.certificates","management.idtags","management.profiles"]'
);
