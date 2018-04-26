package recipe

import (
	"bytes"
	"encoding/json"
	"testing"
)

func _test(original TaskState) bool {
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

	return original == obtained
}

func TestTaskState_JSON(t *testing.T) {
	if !_test(Disabled) {
		t.Errorf("Error testing %s", Disabled)
	}

	if !_test(Enabled) {
		t.Errorf("Error testing %s", Enabled)
	}

	if !_test(Waiting) {
		t.Errorf("Error testing %s", Waiting)
	}

	if !_test(Running) {
		t.Errorf("Error testing %s", Running)
	}

	if !_test(Success) {
		t.Errorf("Error testing %s", Success)
	}

	if !_test(Failure) {
		t.Errorf("Error testing %s", Failure)
	}
}
