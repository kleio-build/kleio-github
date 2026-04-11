package kleiogithub

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShortSHA(t *testing.T) {
	require.Empty(t, shortSHA(""))
	require.Equal(t, "abc", shortSHA("abc"))
	require.Equal(t, "12345678", shortSHA("1234567890"))
}
