package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&User{}, &Account{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// createTestUser creates a test user in the database
func createTestUser(db *gorm.DB) *User {
	user := &User{
		Name:  "Test User",
		Email: "test@example.com",
		Phone: "123-456-7890",
	}
	user.ID = 12345
	db.Create(user)
	return user
}

func TestUserModel(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)

	t.Run("CreateUser", func(t *testing.T) {
		user := User{
			Name:  "John Doe",
			Email: "john@example.com",
			Phone: "123-456-7890",
		}

		err := db.Create(&user).Error
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)
		assert.Equal(t, "John Doe", user.Name)
		assert.Equal(t, "john@example.com", user.Email)
		assert.Equal(t, "123-456-7890", user.Phone)
	})

	t.Run("UniqueEmailConstraint", func(t *testing.T) {
		// Create first user
		user1 := User{
			Name:  "User One",
			Email: "duplicate@example.com",
			Phone: "111-111-1111",
		}
		err := db.Create(&user1).Error
		assert.NoError(t, err)

		// Try to create second user with same email
		user2 := User{
			Name:  "User Two",
			Email: "duplicate@example.com",
			Phone: "222-222-2222",
		}
		err = db.Create(&user2).Error
		assert.Error(t, err) // Should fail due to unique constraint
	})

	t.Run("FindUser", func(t *testing.T) {
		// Create a user
		originalUser := User{
			Name:  "Find Me",
			Email: "findme@example.com",
			Phone: "999-999-9999",
		}
		err := db.Create(&originalUser).Error
		require.NoError(t, err)

		// Find the user
		var foundUser User
		err = db.First(&foundUser, originalUser.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, originalUser.Name, foundUser.Name)
		assert.Equal(t, originalUser.Email, foundUser.Email)
		assert.Equal(t, originalUser.Phone, foundUser.Phone)
	})

	t.Run("UpdateUser", func(t *testing.T) {
		// Create a user
		user := User{
			Name:  "Update Me",
			Email: "updateme@example.com",
			Phone: "555-555-5555",
		}
		err := db.Create(&user).Error
		require.NoError(t, err)

		// Update the user
		user.Name = "Updated Name"
		user.Phone = "777-777-7777"
		err = db.Save(&user).Error
		assert.NoError(t, err)

		// Verify the update
		var updatedUser User
		err = db.First(&updatedUser, user.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", updatedUser.Name)
		assert.Equal(t, "777-777-7777", updatedUser.Phone)
		assert.Equal(t, "updateme@example.com", updatedUser.Email) // Should remain unchanged
	})

	t.Run("DeleteUser", func(t *testing.T) {
		// Create a user
		user := User{
			Name:  "Delete Me",
			Email: "deleteme@example.com",
			Phone: "666-666-6666",
		}
		err := db.Create(&user).Error
		require.NoError(t, err)

		// Delete the user
		err = db.Delete(&user).Error
		assert.NoError(t, err)

		// Verify the user is deleted (soft delete)
		var deletedUser User
		err = db.First(&deletedUser, user.ID).Error
		assert.Error(t, err) // Should not find the user
	})
}

func TestAccountModel(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)

	// Create a user first for foreign key reference
	user := createTestUser(db)

	t.Run("CreateAccount", func(t *testing.T) {
		account := Account{
			UserID:  user.ID,
			Name:    "Savings Account",
			Balance: 1000.50,
		}

		err := db.Create(&account).Error
		assert.NoError(t, err)
		assert.NotZero(t, account.ID)
		assert.Equal(t, user.ID, account.UserID)
		assert.Equal(t, "Savings Account", account.Name)
		assert.Equal(t, 1000.50, account.Balance)
	})

	t.Run("CreateAccountWithDefaultBalance", func(t *testing.T) {
		account := Account{
			UserID: user.ID,
			Name:   "Checking Account",
			// Balance not set, should default to 0.00
		}

		err := db.Create(&account).Error
		assert.NoError(t, err)
		assert.Equal(t, 0.00, account.Balance)
	})

	t.Run("FindAccount", func(t *testing.T) {
		// Create an account
		originalAccount := Account{
			UserID:  user.ID,
			Name:    "Find Account",
			Balance: 500.25,
		}
		err := db.Create(&originalAccount).Error
		require.NoError(t, err)

		// Find the account
		var foundAccount Account
		err = db.First(&foundAccount, originalAccount.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, originalAccount.UserID, foundAccount.UserID)
		assert.Equal(t, originalAccount.Name, foundAccount.Name)
		assert.Equal(t, originalAccount.Balance, foundAccount.Balance)
	})

	t.Run("UpdateAccount", func(t *testing.T) {
		// Create an account
		account := Account{
			UserID:  user.ID,
			Name:    "Update Account",
			Balance: 200.00,
		}
		err := db.Create(&account).Error
		require.NoError(t, err)

		// Update the account
		account.Name = "Updated Account Name"
		account.Balance = 300.75
		err = db.Save(&account).Error
		assert.NoError(t, err)

		// Verify the update
		var updatedAccount Account
		err = db.First(&updatedAccount, account.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, "Updated Account Name", updatedAccount.Name)
		assert.Equal(t, 300.75, updatedAccount.Balance)
		assert.Equal(t, user.ID, updatedAccount.UserID) // Should remain unchanged
	})

	t.Run("DeleteAccount", func(t *testing.T) {
		// Create an account
		account := Account{
			UserID:  user.ID,
			Name:    "Delete Account",
			Balance: 100.00,
		}
		err := db.Create(&account).Error
		require.NoError(t, err)

		// Delete the account
		err = db.Delete(&account).Error
		assert.NoError(t, err)

		// Verify the account is deleted (soft delete)
		var deletedAccount Account
		err = db.First(&deletedAccount, account.ID).Error
		assert.Error(t, err) // Should not find the account
	})

	t.Run("FindAccountsByUser", func(t *testing.T) {
		// Create multiple accounts for the user
		accounts := []Account{
			{UserID: user.ID, Name: "Account 1", Balance: 100.00},
			{UserID: user.ID, Name: "Account 2", Balance: 200.00},
			{UserID: user.ID, Name: "Account 3", Balance: 300.00},
		}

		for _, account := range accounts {
			err := db.Create(&account).Error
			require.NoError(t, err)
		}

		// Find all accounts for the user
		var userAccounts []Account
		err = db.Where("user_id = ?", user.ID).Find(&userAccounts).Error
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(userAccounts), 3) // Should find at least the 3 we created
	})
}
