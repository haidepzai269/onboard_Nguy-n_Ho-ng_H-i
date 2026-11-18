package monitor

import "strconv"

func itoa(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}
