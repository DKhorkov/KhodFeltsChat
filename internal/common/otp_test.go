package common_test

import (
	"testing"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateOTP_InRange(t *testing.T) {
	t.Parallel()

	for range 1000 {
		code, err := common.GenerateOTP()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, code, common.OTPMin)
		assert.LessOrEqual(t, code, common.OTPMax)
	}
}

func TestGenerateOTP_NotAlwaysEqual(t *testing.T) {
	t.Parallel()

	// Sanity check that the generator is not returning a constant.
	// Space is 900000, so 50 samples returning all-equal has probability
	// negligibly close to 0.
	first, err := common.GenerateOTP()
	require.NoError(t, err)

	sawDifferent := false

	for range 50 {
		next, err := common.GenerateOTP()
		require.NoError(t, err)

		if next != first {
			sawDifferent = true

			break
		}
	}

	assert.True(t, sawDifferent, "GenerateOTP should not return the same value repeatedly")
}

func TestGenerateOTP_CoversFullRange(t *testing.T) {
	t.Parallel()

	// With OTPGenerateAttempts=10 covering 900000 possible values, we won't
	// see every value, but we should get a healthy spread over many samples.
	// Verify no sample lands outside the declared range.
	const samples = 5000

	seen := make(map[uint64]struct{}, samples)

	for range samples {
		code, err := common.GenerateOTP()
		require.NoError(t, err)
		require.GreaterOrEqual(t, code, common.OTPMin)
		require.LessOrEqual(t, code, common.OTPMax)

		seen[code] = struct{}{}
	}

	// Birthday paradox: with 5000 samples in a 900000 space, expected unique
	// count is very close to 5000. Requiring >90% unique is extremely safe.
	assert.Greater(t, len(seen), samples*9/10, "OTP should distribute across the range")
}
