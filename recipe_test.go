package recipe

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

/*
Basic usage
*/

func TestRecipe_basic_TOML(t *testing.T) {
	testBasic(t, `
main = "t1"
interp = ['bash', '-c', 'exec {cmd}']

[tasks.t1]
deps = ["t2"]
cmd = "echo t1 >> %[1]s"

[tasks.t2]
deps = ["t3"]
cmd = "echo t2 >> %[1]s"

[tasks.t3]
deps = []
cmd = "echo t3 >> %[1]s"
`, "toml", WarningL)
}

func TestRecipe_basic_JSON(t *testing.T) {
	testBasic(t, `
{
	"main": "t1",
	"interp": ["bash", "-c", "exec {cmd}"],
	"tasks": {
		"t1": {
			"deps": ["t2"],
			"cmd": "echo t1 >> %[1]s",
		},
		"t2": {
			"deps": ["t3"],
			"cmd": "echo t2 >> %[1]s",
		},
		"t3": {
			"deps": [],
			"cmd": "echo t3 >> %[1]s",
		}
	}
}
`, "json", WarningL)
}

func testBasic(t *testing.T, txt, format string, logLevel LoggerLevel) {
	/* Create tmp file */
	name := "basic_output.txt"
	defer os.Remove(name)

	/* Create recipe with tmp file */
	txt = fmt.Sprintf(txt, name)
	fmt.Printf("txt = <<<%s>>>\n", txt)
	path, err := TmpRecipe(format, txt)
	if err != nil {
		t.Errorf("Writing recipe: %s", err)
		return
	}
	defer os.Remove(path)
	defer os.Remove(path + ".state")

	/* Run recipe */
	logger := NewLogger("[Test] ")
	logger.Level = logLevel
	r, err := Open(path, logger, logger)
	if err != nil {
		t.Errorf("Loading recipe: %s", err)
		return
	}
	err = r.RunMain(uint(runtime.NumCPU()))
	if err != nil {
		t.Errorf("Running recipe: %s", err)
		return
	}

	/* Check tmp file */
	data, err := ioutil.ReadFile(name)
	lines := strings.Split(string(data), "\n")
	if !(len(lines) == 4 && lines[0] == "t3" && lines[1] == "t2" && lines[2] == "t1" && lines[3] == "") {
		t.Errorf("Invalid data: %s", data)
	}

	/* Check state */
	if !(r.state.IsSuccess("t1") && r.state.IsSuccess("t2") && r.state.IsSuccess("t3")) {
		t.Errorf("Wrong state: %v", r.state.String())
		return
	}
}

/*
Abort the rest of tasks when any task fail
*/

func TestRecipe_cancel_TOML(t *testing.T) {
	testCancel(t, `
main = "t1"
interp = ['bash', '-c', '{cmd}']

[tasks.t1]
deps = ["t2"]
cmd = "echo t1 >> %[1]s"

[tasks.t2]
deps = ["t3"]
cmd = "false"

[tasks.t3]
deps = []
cmd = "echo t3 >> %[1]s"
`, "toml", WarningL)
}

func TestRecipe_cancel_JSON(t *testing.T) {
	testCancel(t, `
{
	"main": "t1",
	"interp": ["bash", "-c", "{cmd}"],
	"tasks": {
		"t1": {
			"deps": ["t2"],
			"cmd": "echo t1 >> %[1]s",
		},
		"t2": {
			"deps": ["t3"],
			"cmd": "false",
		},
		"t3": {
			"deps": [],
			"cmd": "echo t3 >> %[1]s",
		}
	}
}
`, "json", WarningL)
}

/*
Cancel running tasks when any task fail
*/

func TestRecipe_cancelRunning_TOML(t *testing.T) {
	testCancel(t, `
main = "t1"
interp = ['bash', '-c', '{cmd}']

[tasks.t1]
deps = ["t2", "t3"]
cmd = "echo t1 >> %[1]s"

[tasks.t2]
deps = []
cmd = "sleep 1 && false"

[tasks.t3]
deps = []
cmd = "echo t3 >> %[1]s && sleep 10 && echo FOO >> %[1]s"
`, "toml", DebugL)
}

func TestRecipe_cancelRunning_JSON(t *testing.T) {
	testCancel(t, `
{
	"main": "t1",
	"interp": ["bash", "-c", "{cmd}"],
	"tasks": {
		"t1": {
			"deps": ["t2", "t3"],
			"cmd": "echo t1 >> %[1]s",
		},
		"t2": {
			"deps": [],
			"cmd": "sleep 1 && false",
		},
		"t3": {
			"deps": [],
			"cmd": "echo t3 >> %[1]s && sleep 10 && echo FOO >> %[1]s",
		}
	}
}
`, "json", DebugL)
}

func testCancel(t *testing.T, txt, format string, logLevel LoggerLevel) {
	/* Create tmp file */
	name := "cancel_output.txt"
	defer os.Remove(name)

	/* Create recipe with tmp file */
	txt = fmt.Sprintf(txt, name)
	path, err := TmpRecipe(format, txt)
	if err != nil {
		t.Errorf("Writing recipe: %s", err)
		return
	}
	defer os.Remove(path)
	defer os.Remove(path + ".state")

	/* Run recipe */
	logger := NewLogger("[Test] ")
	logger.Level = logLevel
	r, err := Open(path, logger, logger)
	if err != nil {
		t.Errorf("Loading recipe: %s", err)
		return
	}
	err = r.RunMain(uint(runtime.NumCPU()))
	if err == nil {
		t.Error("Expected failure, not success")
		return
	} else if !strings.Contains(err.Error(), "exit") || !strings.Contains(err.Error(), "t2") {
		t.Error("Expected t2 failure")
		return
	}

	/* Check tmp file */
	data, err := ioutil.ReadFile(name)
	lines := strings.Split(string(data), "\n")
	if !(len(lines) == 2 && lines[0] == "t3" && lines[1] == "") {
		t.Errorf("Invalid data: %s", data)
	}

	/* Check state */
	if !(r.state.IsEnabled("t1") && r.state.IsFailure("t2") && (r.state.IsSuccess("t3") || r.state.IsCancelled("t3"))) {
		t.Errorf("Wrong state: %v", r.state.String())
		return
	}
}

/*
Test utils
*/

func TestRecipe_TmpRecipe(t *testing.T) {
	txt := "Hi world"
	path, err := TmpRecipe("json", txt)
	if err != nil {
		t.Errorf("Writing recipe: %s", err)
		return
	}
	defer os.Remove(path)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("Reading recipe: %s", err)
		return
	}

	if txt != string(data) {
		t.Error("Different content")
		return
	}
}

func TmpRecipe(format, txt string) (string, error) {
	var f *os.File
	var err error
	var path string
	for {
		t := time.Now()
		path = filepath.Join(os.TempDir(), t.Format("20060102150405.999999999")+"."+format)
		f, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0600)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", err
		}
		defer f.Close()
		break
	}
	n, err := f.WriteString(txt)
	if err != nil {
		return "", err
	} else if n < len(txt) {
		return "", io.ErrShortWrite
	}
	return path, nil
}
