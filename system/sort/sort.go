package sort

import (
	"encoding/json"
	"sort"
)

var (
	Strings = sort.Strings
)

func JSON(r json.RawMessage) json.RawMessage {
	var i interface{}

	err := json.Unmarshal(r, &i)
	if err != nil {
		return r
	}

	b, err := json.MarshalIndent(i, "", "    ")
	if err != nil {
		return r
	}
	return b
}
