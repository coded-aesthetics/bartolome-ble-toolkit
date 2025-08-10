package timeular_side_resolver_test

import (
	"testing"
	"timeular_side_resolver"

	"github.com/stretchr/testify/assert"
)

func TestGetSideHighOrLow(t *testing.T) {
	side_low_input := []byte{0x02, 0x00, 0x00}
	side_high_input := []byte{0xfd, 0xff, 0xff}
	invalid_input_1 := []byte{0x02, 0xff, 0x00}
	invalid_input_2 := []byte{0x02, 0xff, 0x00, 0x00}
	invalid_input_3 := []byte{0x02, 0xff}
	invalid_input_4 := []byte{0xfd, 0x00, 0x00}
	invalid_input_5 := []byte{0xff, 0xff, 0xff}
	invalid_input_6 := []byte{0x00, 0x00, 0x00}

	got, err := timeular_side_resolver.GetSideHighOrLow(side_low_input)
	assert.Equal(t, err, nil)
	assert.Equal(t, got, false)

	got, err = timeular_side_resolver.GetSideHighOrLow(side_high_input)
	assert.Equal(t, err, nil)
	assert.Equal(t, got, true)

	_, err = timeular_side_resolver.GetSideHighOrLow(invalid_input_1)
	assert.Equal(t, err.Error(), "this side is not active")

	_, err = timeular_side_resolver.GetSideHighOrLow(invalid_input_2)
	assert.Equal(t, err.Error(), "argument must be of length 3")

	_, err = timeular_side_resolver.GetSideHighOrLow(invalid_input_3)
	assert.Equal(t, err.Error(), "argument must be of length 3")

	_, err = timeular_side_resolver.GetSideHighOrLow(invalid_input_4)
	assert.Equal(t, err.Error(), "this side is not active")

	_, err = timeular_side_resolver.GetSideHighOrLow(invalid_input_5)
	assert.Equal(t, err.Error(), "this side is not active")

	_, err = timeular_side_resolver.GetSideHighOrLow(invalid_input_6)
	assert.Equal(t, err.Error(), "this side is not active")

}

func TestIsIndicatorBitHigh(t *testing.T) {
	for i := 0; i < 256; i++ {
		got := timeular_side_resolver.IsIndicatorBitHigh(byte(i))
		assert.Equal(t, got, i&2 == 2)
	}
}

func TestAreAllBitsHigh(t *testing.T) {
	for i := 0; i < 256; i++ {
		got := timeular_side_resolver.AreAllBitsHigh(byte(i))
		assert.Equal(t, got, i == 255)
	}
}

func TestAreAllBitsLow(t *testing.T) {
	for i := 0; i < 256; i++ {
		got := timeular_side_resolver.AreAllBitsLow(byte(i))
		assert.Equal(t, got, i == 0)
	}
}
