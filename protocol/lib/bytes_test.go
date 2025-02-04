package lib

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUint32ToBytes(t *testing.T) {
	tests := map[string]struct {
		value    uint32
		expected []byte
	}{
		"value of zero": {
			value:    0,
			expected: []byte{0, 0, 0, 0},
		},
		"value of 15": {
			value:    15,
			expected: []byte{0x0f, 0, 0, 0},
		},
		"max uint": {
			// Max uint32 = 4294967295.
			value:    math.MaxUint32,
			expected: []byte{0xff, 0xff, 0xff, 0xff},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := Uint32ToBytes(tc.value)
			require.Equal(t, tc.expected, result)
			require.Equal(t, BytesToUint32(result), tc.value)
		})
	}
}

func TestInt32ToBytes(t *testing.T) {
	tests := map[string]struct {
		value    int32
		expected []byte
	}{
		"value of -1": {
			value:    -1,
			expected: []byte{0xff, 0xff, 0xff, 0xff},
		},
		"value of zero": {
			value:    0,
			expected: []byte{0, 0, 0, 0},
		},
		"value of 15": {
			value:    15,
			expected: []byte{0x0f, 0, 0, 0},
		},
		"max int": {
			// Max int32 = 2147483647.
			value:    math.MaxInt32,
			expected: []byte{0xff, 0xff, 0xff, 0x7f},
		},
		"min int": {
			// Max int32 = -2147483648.
			value:    math.MinInt32,
			expected: []byte{0x00, 0x00, 0x00, 0x80},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := Int32ToBytes(tc.value)
			require.Equal(t, tc.expected, result)
			require.Equal(t, BytesToInt32(result), tc.value)
		})
	}
}

func TestInt64ToBytes(t *testing.T) {
	tests := map[string]struct {
		value    int64
		expected []byte
	}{
		"value of -1": {
			value:    -1,
			expected: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
		"value of zero": {
			value:    0,
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
		"value of 15": {
			value:    15,
			expected: []byte{0x0f, 0, 0, 0, 0, 0, 0, 0},
		},
		"max int": {
			value:    math.MaxInt64,
			expected: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		},
		"min int": {
			value:    math.MinInt64,
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := Int64ToBytes(tc.value)
			require.Equal(t, tc.expected, result)
			require.Equal(t, int64(binary.LittleEndian.Uint64(result)), tc.value)
		})
	}
}

func TestInt32ToString(t *testing.T) {
	i := int32(15)
	require.Equal(t, "15", Int32ToString(i))
}

func TestUint32ToString(t *testing.T) {
	i := uint32(15)
	require.Equal(t, "15", Uint32ToString(i))
}

func TestStringToUint32(t *testing.T) {
	tests := map[string]struct {
		value         string
		expected      uint32
		expectedError string
	}{
		"value of zero": {
			value:    "0",
			expected: uint32(0),
		},
		"value of 100": {
			value:    "100",
			expected: uint32(100),
		},
		"max uint": {
			// Max uint32 = 4294967295.
			value:    "4294967295",
			expected: math.MaxUint32,
		},
		"max uint + 1": {
			// Max uint32 = 4294967295.
			value:         "4294967296",
			expectedError: "value out of range",
		},
		"empty value": {
			value:         "",
			expectedError: "invalid syntax",
		},
		"garbage value": {
			value:         "ffff",
			expectedError: "invalid syntax",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			res, err := StringToUint32(tc.value)
			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, res)
			}
		})
	}
}
