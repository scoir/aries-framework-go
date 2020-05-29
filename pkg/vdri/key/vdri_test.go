/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package key

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/api/vdri"
)

var _ vdri.VDRI = (*VDRI)(nil) // verify interface compliance

func TestAccept(t *testing.T) {
	t.Run("key method", func(t *testing.T) {
		v, err := New()
		require.NoError(t, err)
		require.NotNil(t, v)

		accept := v.Accept("key")
		require.True(t, accept)
	})

	t.Run("other method", func(t *testing.T) {
		v, err := New()
		require.NoError(t, err)
		require.NotNil(t, v)

		accept := v.Accept("other")
		require.False(t, accept)
	})
}

func TestStore(t *testing.T) {
	t.Run("test success", func(t *testing.T) {
		v, err := New()
		require.NoError(t, err)
		require.NoError(t, v.Store(nil, nil))
	})
}

func TestClose(t *testing.T) {
	t.Run("test success", func(t *testing.T) {
		v, err := New()
		require.NoError(t, err)
		require.NoError(t, v.Close())
	})
}
