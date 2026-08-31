package cli

import (
	"fmt"
	"strconv"
	"strings"
)

func treePath(key string, structured bool) ([]interface{}, error) {
	if !structured {
		return []interface{}{key}, nil
	}
	var path []interface{}
	for i := 0; i < len(key); {
		start := i
		for i < len(key) && key[i] != '.' && key[i] != '[' {
			i++
		}
		if start == i {
			return nil, fmt.Errorf("invalid key path %q", key)
		}
		path = append(path, key[start:i])
		for i < len(key) && key[i] == '[' {
			end := strings.IndexByte(key[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("invalid key path %q", key)
			}
			end += i
			index, err := strconv.Atoi(key[i+1 : end])
			if err != nil || index < 0 {
				return nil, fmt.Errorf("invalid key path %q", key)
			}
			path = append(path, index)
			i = end + 1
		}
		if i < len(key) {
			if key[i] != '.' {
				return nil, fmt.Errorf("invalid key path %q", key)
			}
			i++
		}
	}
	return path, nil
}
