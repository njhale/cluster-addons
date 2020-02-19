package builder

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func BuildObject(template string, in string) (out []string, err error) {
	if !gjson.Valid(in) {
		err = fmt.Errorf("not valid json")
		return
	}
	out = []string{""}
	parsed := gjson.Get(template, "@this")
	parsed.ForEach(func(key, value gjson.Result) bool {
		inline := false
		query := value.String()
		if value.IsArray() && len(value.Array()) == 1 {
			inline = true
			query = value.Array()[0].String()
		}
		fetched := gjson.Get(in, query)
		if !inline && fetched.IsArray() {
			newOut := make([]string, 0)
			for _, val := range fetched.Array() {
				for _, o := range out {
					gened, err := sjson.SetRaw(o, key.String(), val.Raw)
					if err != nil {
						return false
					}
					newOut = append(newOut, gened)
				}
				out = newOut
			}
			return true
		}
		for i, o := range out {
			out[i], err = sjson.SetRaw(o, key.String(), fetched.Raw)
			if err != nil {
				return false
			}
		}
		return true
	})
	return
}
