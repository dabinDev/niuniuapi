//go:build unit

package dto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseUserVisibleMenuItemsRequiresExplicitUserVisibility(t *testing.T) {
	raw := `[
		{"id":"legacy","label":"Legacy","url":"https://legacy.example","sort_order":0},
		{"id":"admin","label":"Admin","url":"https://admin.example","visibility":"admin","sort_order":1},
		{"id":"user","label":"User","url":"https://user.example","visibility":"user","sort_order":2}
	]`

	items := ParseUserVisibleMenuItems(raw)

	require.Len(t, items, 1)
	require.Equal(t, "user", items[0].ID)
	require.Equal(t, "user", items[0].Visibility)
}
