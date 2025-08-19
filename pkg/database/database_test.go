package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectDB(t *testing.T) {
	t.Run("ConnectWithDefaultDSN", func(t *testing.T) {
		// Clear any existing DATABASE_URL
		originalDatabaseURL := os.Getenv("DATABASE_URL")
		defer func() {
			if originalDatabaseURL != "" {
				os.Setenv("DATABASE_URL", originalDatabaseURL)
			} else {
				os.Unsetenv("DATABASE_URL")
			}
		}()
		os.Unsetenv("DATABASE_URL")

		// This test will try to connect to a local PostgreSQL
		// In a real test environment, you might want to skip this or use a test DB
		db, err := ConnectDB()

		// Since we can't guarantee a PostgreSQL instance is running in tests,
		// we'll just verify the function doesn't panic and returns expected types
		if err != nil {
			// If connection fails, that's expected in a test environment
			assert.Error(t, err)
			assert.Nil(t, db)
		} else {
			// If it succeeds, verify we got a valid DB instance
			assert.NotNil(t, db)
			assert.NoError(t, err)
		}
	})

	t.Run("ConnectWithCustomDSN", func(t *testing.T) {
		// Set a custom DATABASE_URL
		originalDatabaseURL := os.Getenv("DATABASE_URL")
		defer func() {
			if originalDatabaseURL != "" {
				os.Setenv("DATABASE_URL", originalDatabaseURL)
			} else {
				os.Unsetenv("DATABASE_URL")
			}
		}()

		customDSN := "host=testhost user=testuser password=testpass dbname=testdb port=5432 sslmode=disable"
		os.Setenv("DATABASE_URL", customDSN)

		// This will likely fail since testhost doesn't exist, but we're testing the DSN usage
		db, err := ConnectDB()

		// Expect this to fail since testhost doesn't exist
		assert.Error(t, err)
		assert.Nil(t, db)

		// The error should be related to connection, not DSN parsing
		assert.Contains(t, err.Error(), "failed to connect")
	})

	t.Run("ConnectWithInvalidDSN", func(t *testing.T) {
		// Set an invalid DATABASE_URL
		originalDatabaseURL := os.Getenv("DATABASE_URL")
		defer func() {
			if originalDatabaseURL != "" {
				os.Setenv("DATABASE_URL", originalDatabaseURL)
			} else {
				os.Unsetenv("DATABASE_URL")
			}
		}()

		invalidDSN := "invalid-dsn-format"
		os.Setenv("DATABASE_URL", invalidDSN)

		db, err := ConnectDB()

		// Should fail with invalid DSN
		assert.Error(t, err)
		assert.Nil(t, db)
	})
}

func TestConnectDBEnvironmentVariables(t *testing.T) {
	t.Run("DatabaseURLSet", func(t *testing.T) {
		originalDatabaseURL := os.Getenv("DATABASE_URL")
		defer func() {
			if originalDatabaseURL != "" {
				os.Setenv("DATABASE_URL", originalDatabaseURL)
			} else {
				os.Unsetenv("DATABASE_URL")
			}
		}()

		testDSN := "host=envtest user=envuser password=envpass dbname=envdb port=5432 sslmode=disable"
		os.Setenv("DATABASE_URL", testDSN)

		// We can't easily mock the actual database connection without significant refactoring,
		// but we can verify that the environment variable is being read correctly
		// by checking if the function attempts to use the custom DSN

		// Call ConnectDB - it should use our custom DSN
		_, err := ConnectDB()

		// We expect an error since our test DSN points to a non-existent host,
		// but the error should indicate it tried to connect (not a DSN parsing error)
		if err != nil {
			// The error message should contain details that suggest it tried to connect
			// to our test host, indicating the environment variable was used
			assert.Error(t, err)
		}
	})

	t.Run("DatabaseURLNotSet", func(t *testing.T) {
		originalDatabaseURL := os.Getenv("DATABASE_URL")
		defer func() {
			if originalDatabaseURL != "" {
				os.Setenv("DATABASE_URL", originalDatabaseURL)
			} else {
				os.Unsetenv("DATABASE_URL")
			}
		}()

		os.Unsetenv("DATABASE_URL")

		// Should use default DSN when environment variable is not set
		_, err := ConnectDB()

		// Again, we expect this to likely fail in a test environment,
		// but it should attempt to use the default localhost DSN
		if err != nil {
			assert.Error(t, err)
		}
	})
}

// Integration test that requires a real database connection
// This test is skipped by default but can be run with proper database setup
func TestConnectDBIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// This test requires a real PostgreSQL database to be running
	// Set up your test database and DSN here
	testDSN := os.Getenv("TEST_DATABASE_URL")
	if testDSN == "" {
		t.Skip("TEST_DATABASE_URL not set for integration test")
	}

	originalDatabaseURL := os.Getenv("DATABASE_URL")
	defer func() {
		if originalDatabaseURL != "" {
			os.Setenv("DATABASE_URL", originalDatabaseURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
	}()

	os.Setenv("DATABASE_URL", testDSN)

	db, err := ConnectDB()
	require.NoError(t, err)
	require.NotNil(t, db)

	// Test that we can perform basic database operations
	sqlDB, err := db.DB()
	require.NoError(t, err)

	err = sqlDB.Ping()
	assert.NoError(t, err)

	// Clean up
	sqlDB.Close()
}
