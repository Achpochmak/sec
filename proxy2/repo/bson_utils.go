package repo

import (
	"go.mongodb.org/mongo-driver/bson"
)

func convertToBSON(values map[string][]string) bson.M {
	result := make(bson.M, len(values))

	for key, vals := range values {
		if len(vals) == 1 {
			result[key] = vals[0]
		} else {
			result[key] = vals
		}
	}

	return result
}

func convertFromBSON(data bson.M) map[string][]string {
	result := make(map[string][]string, len(data))

	for key, value := range data {
		switch v := value.(type) {
		case string:
			result[key] = []string{v}
		case []interface{}:
			result[key] = convertToStringSlice(v)
		}
	}

	return result
}

func convertToStringSlice(arr []interface{}) []string {
	result := make([]string, 0, len(arr))

	for _, elem := range arr {
		if str, ok := elem.(string); ok {
			result = append(result, str)
		}
	}

	return result
}
