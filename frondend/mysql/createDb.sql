-- 1. 角色与功能权限表 (对应 5.1 和 5.2)
CREATE TABLE `roles` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `role_name` VARCHAR(50) NOT NULL, -- Admin, CP-OP, CP-User
    `permissions` JSON, -- 存储该角色被授权的模块，如 ["overview", "ocpp_config", "pnc"]
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- 2. 用户表 (对应 2.3.4.1)
CREATE TABLE `users` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `username` VARCHAR(100) NOT NULL UNIQUE,
    `password_hash` VARCHAR(255) NOT NULL,
    `email` VARCHAR(100),
    `contact_info` VARCHAR(100),
    `company_name` VARCHAR(100),
    `role_id` INT NOT NULL,
    `parent_user_id` INT DEFAULT NULL, -- 用于绑定 CP-User 到特定的 CP-OP
    `is_enabled` BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (`role_id`) REFERENCES `roles`(`id`),
    FOREIGN KEY (`parent_user_id`) REFERENCES `users`(`id`)
) ENGINE=InnoDB;

-- 3. 设备表 (对应 2.3.4.3 & 5.4)
CREATE TABLE `devices` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `device_name` VARCHAR(100) NOT NULL,
    `charge_point_id` VARCHAR(100) NOT NULL UNIQUE,
    `secc_id` VARCHAR(255) UNIQUE COMMENT 'ISO 15118-20 固定的 SECCID', 
    `protocol` VARCHAR(50), -- OCPP 1.6J / 2.0.1
    `location` VARCHAR(255),
    `heartbeat_interval` INT DEFAULT 300,
    `is_enabled` BOOLEAN DEFAULT TRUE,
    `owner_user_id` INT NOT NULL, -- 归属的 CP-User
    FOREIGN KEY (`owner_user_id`) REFERENCES `users`(`id`)
) ENGINE=InnoDB;

-- 4. 交易订单表 (对应 2.3.2.1 bill & 5.3)
CREATE TABLE `transactions` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `transaction_id` INT NOT NULL UNIQUE,
    `charge_point_id` VARCHAR(100) NOT NULL,
    `connector_id` INT NOT NULL,
    `start_time` DATETIME NOT NULL,
    `stop_time` DATETIME,
    `duration` INT, -- 可以用触发器或代码自动计算
    `start_meter` DECIMAL(10, 2),
    `stop_meter` DECIMAL(10, 2),
    `cost_energy` DECIMAL(10, 2),
    `status` VARCHAR(50) -- Running, Completed, Faulted
) ENGINE=InnoDB;

-- 5. 证书管理表 (对应 2.3.4.2 & 5.5)
CREATE TABLE `certificates` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `file_name` VARCHAR(255) NOT NULL,
    `cert_type` VARCHAR(50) NOT NULL, -- Mo root, V2G root, SECC Leaf 等
    `file_path` VARCHAR(500) NOT NULL,
    `uploaded_by` INT NOT NULL, -- 仅 Admin 和 CP-OP
    `upload_time` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`uploaded_by`) REFERENCES `users`(`id`)
) ENGINE=InnoDB;