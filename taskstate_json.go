package recipe

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (s TaskState) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *TaskState) UnmarshalJSON(b []byte) error {
	b = b[1 : len(b)-1]
	position := strings.Index(_TaskState_name, string(b))
	for i, p := range _TaskState_index {
		if position == int(p) {
			*s = TaskState(i)
			return nil
		}
	}
	return fmt.Errorf("Unknown TaskState: %s", b)
}
