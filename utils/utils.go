package utils

func Filter_Array[K any](data []K, f func(K) bool) []K {
	fltd := make([]K, 0)

	for _, e := range data {

		if f(e) {
			fltd = append(fltd, e)
		}
	}

	return fltd
}
