package recipe

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestTaskState_EncodeAndDecode(t *testing.T) {
	_testEncodeAndDecode := func(t *testing.T, original TaskState) {
		b := bytes.Buffer{}
		e := json.NewEncoder(&b)
		err := e.Encode(original)
		if err != nil {
			panic(err)
		}
		var obtained TaskState
		err = json.NewDecoder(&b).Decode(&obtained)
		if err != nil {
			panic(err)
		}

		if original != obtained {
			t.Errorf("Error testing %s", original)
		}
	}
	_testEncodeAndDecode(t, Disabled)
	_testEncodeAndDecode(t, Enabled)
	_testEncodeAndDecode(t, Waiting)
	_testEncodeAndDecode(t, Running)
	_testEncodeAndDecode(t, Success)
	_testEncodeAndDecode(t, Failure)
}
