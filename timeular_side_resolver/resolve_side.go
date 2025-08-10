package timeular_side_resolver

import "errors"

func Resolve_Side(payload []byte) (byte, error) {
	if len(payload) != 12 {
		return 0, errors.New("payload must be a byte array of length 12")
	}
	first := payload[1:4]
	second := payload[5:8]
	third := payload[9:]

	first_high_or_low, err_first := GetSideHighOrLow(first)
	second_high_or_low, err_second := GetSideHighOrLow(second)
	third_high_or_low, err_third := GetSideHighOrLow(third)

	side := byte(0)

	if err_first != nil && err_second != nil {
		return 0, errors.New("side information could not be determined")
	}

	if err_first == nil {
		if first_high_or_low {
			side += 2
		} else {
			side += 4
		}
	}

	if err_second == nil {
		if second_high_or_low {
			side += 6
		}
	}

	if err_third == nil {
		if third_high_or_low {
			side += 1
		}
	} else {
		return 0, errors.New("side information could not be determined")
	}

	return side, nil
}
