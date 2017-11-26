package main

import (
	"dev"
	"fmt"
	"msg"
	"mylog"
)

func main() {
	runinfo := new(dev.RunInfo)
	//fmt.Println(reflect.TypeOf(runinfo))
	runinfo.CmdQ = make(chan string, 100)
	//fmt.Println(reflect.TypeOf(runinfo.CmdQ))
	runinfo.Config = dev.ReadConfigFile("config.xml")
	runinfo.InitPeer()
	msg.Init(runinfo)
	go msg.ReciveMsg(runinfo.Cmdconn, runinfo.CmdQ)
	go msg.ProcessMsg(runinfo.CmdQ, runinfo)
	go msg.LogForPlay(runinfo.Logconn)
	ver := "Welcom to Device Center. ver:0.10"
	mylog.RecordLog(ver)
	var a int
	fmt.Scanf("%d", &a)
}
