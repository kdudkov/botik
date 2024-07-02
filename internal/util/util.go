package util

const TIME_FMT = "02.01.2006 15:04"

func IsInArray(str string, array ...string) bool {
	for _, s1 := range array {
		if str == s1 {
			return true
		}
	}

	return false
}

func HasAllKeys(m map[string]interface{}, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}

	return true
}
