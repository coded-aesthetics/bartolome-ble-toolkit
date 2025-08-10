package timeular_side_resolver

import "errors"

func GetSideHighOrLow(sideBytes []byte) (bool, error) {
	if len(sideBytes) != 3 {
		return false, errors.New("argument must be of length 3")
	}
	is_indicator_bit_high := IsIndicatorBitHigh(sideBytes[0])
	if is_indicator_bit_high {
		are_all_other_bits_low := AreAllBitsLow(sideBytes[1]) && AreAllBitsLow(sideBytes[2])

		if are_all_other_bits_low {
			return false, nil
		}
	} else {
		are_all_other_bits_high := AreAllBitsHigh(sideBytes[1]) && AreAllBitsHigh(sideBytes[2])
		if are_all_other_bits_high {
			return true, nil
		}
	}
	return false, errors.New("this side is not active")
}

// checks whether the second last bit is high (1)
func IsIndicatorBitHigh(oneByte byte) bool {
	// 0011 & 0010 == 0010
	// 0010 & 0010 == 0010
	return oneByte&2 == 2
}

func AreAllBitsHigh(oneByte byte) bool {
	return oneByte == 0xff
}

func AreAllBitsLow(oneByte byte) bool {
	return oneByte == 0
}
