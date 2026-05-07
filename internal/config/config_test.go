package config

import (
	"os"
	"testing"
)

// TestLoad 测试加载配置
func TestLoad(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("SERVER_ENV", "development")
	os.Setenv("PUBLIC_BASE_URL", "http://localhost:8080")
	os.Setenv("ADMIN_API_KEY", "test_admin_key")
	os.Setenv("ADMIN_IP_WHITELIST", "127.0.0.1,::1")
	os.Setenv("MASTER_KEY", "test_master_key")
	os.Setenv("DB_PASSWORD", "test_password")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_NAME", "test_db")
	defer func() {
		os.Unsetenv("SERVER_ENV")
		os.Unsetenv("PUBLIC_BASE_URL")
		os.Unsetenv("ADMIN_API_KEY")
		os.Unsetenv("ADMIN_IP_WHITELIST")
		os.Unsetenv("MASTER_KEY")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_NAME")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.Password != "test_password" {
		t.Errorf("Password = %v, want test_password", cfg.Database.Password)
	}
	if cfg.Database.User != "test_user" {
		t.Errorf("User = %v, want test_user", cfg.Database.User)
	}
	if cfg.Database.DBName != "test_db" {
		t.Errorf("DBName = %v, want test_db", cfg.Database.DBName)
	}
	if cfg.PublicBaseURL != "http://localhost:8080" {
		t.Errorf("PublicBaseURL = %v, want http://localhost:8080", cfg.PublicBaseURL)
	}
	if cfg.AdminAPIKey != "test_admin_key" {
		t.Errorf("AdminAPIKey = %v, want test_admin_key", cfg.AdminAPIKey)
	}
	if cfg.AdminIPWhitelist != "127.0.0.1,::1" {
		t.Errorf("AdminIPWhitelist = %v, want 127.0.0.1,::1", cfg.AdminIPWhitelist)
	}
}

// TestLoad_ProductionMissingMasterKey 测试生产环境缺少 MASTER_KEY
func TestLoad_ProductionMissingMasterKey(t *testing.T) {
	os.Setenv("SERVER_ENV", "production")
	os.Setenv("PUBLIC_BASE_URL", "https://pay.example.com")
	os.Setenv("ADMIN_API_KEY", "test_admin_key")
	os.Setenv("DB_PASSWORD", "test_password")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_NAME", "test_db")
	os.Unsetenv("MASTER_KEY")
	defer func() {
		os.Unsetenv("SERVER_ENV")
		os.Unsetenv("PUBLIC_BASE_URL")
		os.Unsetenv("ADMIN_API_KEY")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_NAME")
	}()

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error when MASTER_KEY is missing in production")
	}
}

// TestLoad_MissingPassword 测试缺少密码
func TestLoad_MissingPassword(t *testing.T) {
	// 清除密码环境变量
	os.Setenv("SERVER_ENV", "development")
	os.Unsetenv("DB_PASSWORD")
	defer os.Unsetenv("SERVER_ENV")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when DB_PASSWORD is missing")
	}
}

// TestValidate 测试配置验证
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				ServerPort: "8080",
				Database: DatabaseConfig{
					User:     "test_user",
					Password: "test_password",
					DBName:   "test_db",
				},
			},
			wantErr: false,
		},
		{
			name: "missing password",
			cfg: &Config{
				ServerPort: "8080",
				Database: DatabaseConfig{
					User:   "test_user",
					DBName: "test_db",
				},
			},
			wantErr: true,
		},
		{
			name: "missing user",
			cfg: &Config{
				ServerPort: "8080",
				Database: DatabaseConfig{
					Password: "test_password",
					DBName:   "test_db",
				},
			},
			wantErr: true,
		},
		{
			name: "missing db name",
			cfg: &Config{
				ServerPort: "8080",
				Database: DatabaseConfig{
					User:     "test_user",
					Password: "test_password",
				},
			},
			wantErr: true,
		},
		{
			name: "production with insecure admin config",
			cfg: &Config{
				ServerPort:    "8080",
				ServerEnv:     "production",
				PublicBaseURL: "http://localhost:8080",
				AdminAPIKey:   "default-insecure-key-change-me",
				Database: DatabaseConfig{
					User:     "test_user",
					Password: "test_password",
					DBName:   "test_db",
				},
			},
			wantErr: true,
		},
		{
			name: "production config ok",
			cfg: &Config{
				ServerPort:    "8080",
				ServerEnv:     "production",
				PublicBaseURL: "https://pay.example.com",
				AdminAPIKey:   "secure_admin_key",
				Database: DatabaseConfig{
					User:     "test_user",
					Password: "test_password",
					DBName:   "test_db",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGetEnv 测试获取环境变量
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "env exists",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "env not exists",
			key:          "TEST_KEY_NOT_EXISTS",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetEnvInt 测试获取整数环境变量
func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		want         int
	}{
		{
			name:         "valid int",
			key:          "TEST_INT",
			defaultValue: 100,
			envValue:     "200",
			want:         200,
		},
		{
			name:         "invalid int",
			key:          "TEST_INT_INVALID",
			defaultValue: 100,
			envValue:     "invalid",
			want:         100,
		},
		{
			name:         "env not exists",
			key:          "TEST_INT_NOT_EXISTS",
			defaultValue: 100,
			envValue:     "",
			want:         100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnvInt(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
