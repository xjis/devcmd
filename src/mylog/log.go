package mylog

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

/*
获取程序运行路径
*/
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return ""
	}
	//return strings.Replace(dir, "\\", "/", -1)
	return dir
}

/*
记录日志
*/
func RecordLog(msg string) {
	wirteToFile(msg, "DevLog", 1)
}

func RecordPlayLog(msg string) {
	wirteToFile(msg, "PlayLog", 0)
}

//.../log/yyyymm/DevLogyyyymmdd.log
//.../log/yyyymm/PlayLogyyyymmdd.log
func wirteToFile(msg string, name string, tag int) {
	t := time.Now()
	s := fmt.Sprintf("%04d%02d", int(t.Year()), int(t.Month()))
	logdir := getCurrentDirectory() + "\\log" + "\\" + s
	_ = os.MkdirAll(logdir, os.ModeDir)
	file := logdir + "\\" + name + fmt.Sprintf("%04d%02d%02d", int(t.Year()), int(t.Month()), int(t.Day())) + ".log"
	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND, os.ModeAppend)
	defer f.Close()
	if err == nil {
		s := t.Format("2006-01-02 15:04:05") + " " + msg + "\r\n"
		f.Write([]byte(s))
		if tag == 1 {
			fmt.Print(s)
		}
	}
}
