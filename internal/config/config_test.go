package config

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func validEnv() map[string]string {
	return map[string]string{
		"DB_HOST":     "localhost",
		"DB_PORT":     "3306",
		"DB_NAME":     "testdb",
		"DB_USER":     "root",
		"DB_PASSWORD": "secret",
		"API_KEYS":    "key1,key2",
	}
}

func TestLoad_Success(t *testing.T) {
	setEnv(t, validEnv())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DBHost != "localhost" {
		t.Errorf("DBHost = %q, want %q", cfg.DBHost, "localhost")
	}
	if cfg.DBPort != "3306" {
		t.Errorf("DBPort = %q, want %q", cfg.DBPort, "3306")
	}
	if cfg.DBName != "testdb" {
		t.Errorf("DBName = %q, want %q", cfg.DBName, "testdb")
	}
	if cfg.DBUser != "root" {
		t.Errorf("DBUser = %q, want %q", cfg.DBUser, "root")
	}
	if cfg.DBPassword != "secret" {
		t.Errorf("DBPassword = %q, want %q", cfg.DBPassword, "secret")
	}
	if len(cfg.APIKeys) != 2 || cfg.APIKeys[0] != "key1" || cfg.APIKeys[1] != "key2" {
		t.Errorf("APIKeys = %v, want [key1 key2]", cfg.APIKeys)
	}
	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":8080")
	}
}

func TestLoad_CustomListenAddr(t *testing.T) {
	env := validEnv()
	env["LISTEN_ADDR"] = ":9090"
	setEnv(t, env)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":9090")
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	required := []string{"DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD", "API_KEYS"}
	for _, key := range required {
		t.Run("missing_"+key, func(t *testing.T) {
			env := validEnv()
			delete(env, key)
			setEnv(t, env)

			_, err := Load()
			if err == nil {
				t.Errorf("expected error when %s is missing", key)
			}
		})
	}
}

func TestLoad_EmptyAPIKeys(t *testing.T) {
	env := validEnv()
	env["API_KEYS"] = ",,,"
	setEnv(t, env)

	_, err := Load()
	if err == nil {
		t.Error("expected error when API_KEYS contains only commas")
	}
}

func TestLoad_SingleAPIKey(t *testing.T) {
	env := validEnv()
	env["API_KEYS"] = "onlykey"
	setEnv(t, env)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.APIKeys) != 1 || cfg.APIKeys[0] != "onlykey" {
		t.Errorf("APIKeys = %v, want [onlykey]", cfg.APIKeys)
	}
}

// Feature: mariadb-dal-api, Property 1: Config parsing round-trip
// Validates: Requirements 1.1, 2.1
func TestPropertyConfigRoundTrip(t *testing.T) {
	envVars := []string{"DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD", "API_KEYS", "LISTEN_ADDR"}

	rapid.Check(t, func(rt *rapid.T) {
		host := rapid.StringMatching(`[a-zA-Z0-9._-]+`).Draw(rt, "host")
		port := rapid.StringMatching(`[0-9]{1,5}`).Draw(rt, "port")
		dbname := rapid.StringMatching(`[a-zA-Z0-9_]+`).Draw(rt, "dbname")
		user := rapid.StringMatching(`[a-zA-Z0-9_]+`).Draw(rt, "user")
		password := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*_-]+`).Draw(rt, "password")

		numKeys := rapid.IntRange(1, 5).Draw(rt, "numKeys")
		keys := make([]string, numKeys)
		for i := range keys {
			keys[i] = rapid.StringMatching(`[a-zA-Z0-9_-]+`).Draw(rt, fmt.Sprintf("key%d", i))
		}
		apiKeysEnv := strings.Join(keys, ",")

		listenAddr := rapid.StringMatching(`:[0-9]{2,5}`).Draw(rt, "listenAddr")

		// Set env vars and register cleanup via rt.Cleanup
		vals := map[string]string{
			"DB_HOST":     host,
			"DB_PORT":     port,
			"DB_NAME":     dbname,
			"DB_USER":     user,
			"DB_PASSWORD": password,
			"API_KEYS":    apiKeysEnv,
			"LISTEN_ADDR": listenAddr,
		}
		for _, k := range envVars {
			os.Setenv(k, vals[k])
		}
		rt.Cleanup(func() {
			for _, k := range envVars {
				os.Unsetenv(k)
			}
		})

		cfg, err := Load()
		if err != nil {
			rt.Fatalf("Load() returned unexpected error: %v", err)
		}

		if cfg.DBHost != host {
			rt.Errorf("DBHost = %q, want %q", cfg.DBHost, host)
		}
		if cfg.DBPort != port {
			rt.Errorf("DBPort = %q, want %q", cfg.DBPort, port)
		}
		if cfg.DBName != dbname {
			rt.Errorf("DBName = %q, want %q", cfg.DBName, dbname)
		}
		if cfg.DBUser != user {
			rt.Errorf("DBUser = %q, want %q", cfg.DBUser, user)
		}
		if cfg.DBPassword != password {
			rt.Errorf("DBPassword = %q, want %q", cfg.DBPassword, password)
		}
		if len(cfg.APIKeys) != len(keys) {
			rt.Errorf("APIKeys length = %d, want %d", len(cfg.APIKeys), len(keys))
		} else {
			for i, k := range keys {
				if cfg.APIKeys[i] != k {
					rt.Errorf("APIKeys[%d] = %q, want %q", i, cfg.APIKeys[i], k)
				}
			}
		}
		if cfg.ListenAddr != listenAddr {
			rt.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, listenAddr)
		}
	})
}
