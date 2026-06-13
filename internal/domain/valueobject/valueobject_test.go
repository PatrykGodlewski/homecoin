package valueobject_test

import (
	"testing"

	"github.com/godlew/homecoin/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmail(t *testing.T) {
	t.Run("valid email is normalized", func(t *testing.T) {
		email, err := valueobject.NewEmail("  Alice@Example.COM ")
		require.NoError(t, err)
		assert.Equal(t, "alice@example.com", email.String())
	})

	t.Run("invalid email", func(t *testing.T) {
		_, err := valueobject.NewEmail("not-an-email")
		assert.Error(t, err)
	})
}

func TestNewMoney(t *testing.T) {
	t.Run("valid money", func(t *testing.T) {
		m, err := valueobject.NewMoney(1500, "usd")
		require.NoError(t, err)
		assert.Equal(t, int64(1500), m.AmountCents)
		assert.Equal(t, "USD", m.Currency)
	})

	t.Run("negative amount", func(t *testing.T) {
		_, err := valueobject.NewMoney(-1, "USD")
		assert.Error(t, err)
	})

	t.Run("invalid currency", func(t *testing.T) {
		_, err := valueobject.NewMoney(100, "US")
		assert.Error(t, err)
	})
}

func TestParseSplitType(t *testing.T) {
	valid := []string{"equal", "exact", "percentage", "shares"}
	for _, s := range valid {
		st, err := valueobject.ParseSplitType(s)
		require.NoError(t, err)
		assert.Equal(t, valueobject.SplitType(s), st)
	}

	_, err := valueobject.ParseSplitType("invalid")
	assert.Error(t, err)
}
