package utils

//TODO
func IsAnyEmpty(ss ...interface{}) bool {
	for _, v := range ss {
		if v.(string) == "" {
			return true
		}
	}
	return false
}
