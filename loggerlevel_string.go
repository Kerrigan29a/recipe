// Code generated by "stringer -type=LoggerLevel"; DO NOT EDIT.

package recipe

import "strconv"

const _LoggerLevel_name = "DebugLInfoLWarningLErrorLFatalL"

var _LoggerLevel_index = [...]uint8{0, 6, 11, 19, 25, 31}

func (i LoggerLevel) String() string {
	if i < 0 || i >= LoggerLevel(len(_LoggerLevel_index)-1) {
		return "LoggerLevel(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _LoggerLevel_name[_LoggerLevel_index[i]:_LoggerLevel_index[i+1]]
}
