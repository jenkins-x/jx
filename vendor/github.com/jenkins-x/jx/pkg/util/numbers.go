package util

import "strconv"

func Int32ToA(n int32) string {
	return strconv.FormatInt(int64(n), 10)
}

func AtoInt32(text string) (int32, error) {
	i, err := strconv.Atoi(text)
	if err != nil {
		return 0, err
	}
	return int32(i), nil
}
