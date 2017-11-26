// msg
package msg

import (
	"dev"
	"fmt"
	"mylog"
	"net"
	"strings"
)

func Init(runinfo *dev.RunInfo) {
	service := ":" + runinfo.Config.Serverdata.Server[0].Port
	udpAddr, err := net.ResolveUDPAddr("udp4", service)
	if err != nil {
		mylog.RecordLog(err.Error())
		return
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		mylog.RecordLog(err.Error())
		return
	}
	runinfo.Cmdconn = conn

	logservice := ":10010"
	udpAddr, err = net.ResolveUDPAddr("udp4", logservice)
	if err != nil {
		mylog.RecordLog(err.Error())
		return
	}
	conn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		mylog.RecordLog(err.Error())
		return
	}
	runinfo.Logconn = conn
	d := len(runinfo.Config.Devicedata.Dev)
	p := len(runinfo.Config.Comdata.Com)
	log := fmt.Sprintf("Init complete. Total %d devices , %d peers ", d, p)
	mylog.RecordLog(log)
	return
}

func ReciveMsg(conn *net.UDPConn, c chan string) {
	var buf [1024]byte
	for {
		n, rAddr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			mylog.RecordLog(err.Error())
			return
		}
		log := fmt.Sprintf("Recv msg from %v %v", rAddr.String(), string(buf[0:n]))
		mylog.RecordLog(log)
		c <- string(buf[0:n])
	}
}
func LogForPlay(conn *net.UDPConn) {
	var buf [1024]byte
	for {
		n, rAddr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			return
		}
		log := fmt.Sprintf("Recv msg from %v %v", rAddr.String(), string(buf[0:n]))
		mylog.RecordPlayLog(log)
	}
}
func ProcessMsg(c chan string, runinfo *dev.RunInfo) {
	for {
		currentmsg, ok := <-c
		if !ok {
			continue
		}
		//fmt.Println(runinfo.DevPeer[0])
		msgs := strings.Split(currentmsg, ";")
		for _, msg := range msgs {
			if len(msg) < 3 {
				//规避会打印一个空出来
				continue
			}
			cmds := strings.Split(msg, ",")
			if len(cmds) < 3 {
				log := "error cmd:" + msg
				mylog.RecordLog(log)
				continue
			}
			f := getProcessfun(cmds[0])
			if f == nil {
				continue
			}
			runinfo.CurMsg = msg
			cmd3 := ""
			if len(cmds) >= 4 {
				cmd3 = cmds[3]
			}
			f(cmds[1], cmds[2], cmd3, runinfo)
		}
		//fmt.Println(runinfo.DevPeer[0])
		//fmt.Println(runinfo)
	}
}
func getProcessfun(cmd string) func(string, string, string, *dev.RunInfo) {
	switch strings.ToLower(cmd) {
	case "device_ctrl":
		return processSwitch
	case "matrix_ctrl":
		return processMatrix
	default:
		return nil
	}

}
func processSwitch(id string, cmd string, pad string, runinfo *dev.RunInfo) {
	for i, _ := range runinfo.Config.Devicedata.Dev {
		d := &runinfo.Config.Devicedata.Dev[i]
		if d.Dtype != dev.SWITCH {
			continue
		}
		if d.Id == id {
			cmdtodev := d.Cmd1
			if strings.ToLower(cmd) == "close" {
				cmdtodev = d.Cmd2
			}
			for i, _ := range runinfo.DevPeer {
				p := &runinfo.DevPeer[i]
				if p.ConfigInfo.Id == d.Com {
					//fmt.Println(p)
					//dbg := "DBGINFOB:cmdtodev:" + cmdtodev + " p.CmdBuff:" + p.CmdBuff
					//fmt.Println(dbg)
					//p.CmdBuff += cmdtodev
					//var buffer bytes.Buffer
					//buffer.WriteString(p.CmdBuff)
					//buffer.WriteString(cmdtodev)
					//p.CmdBuff = buffer.String()
					//dbg = "DBGINFOA:cmdtodev:" + cmdtodev + " p.CmdBuff:" + p.CmdBuff
					//fmt.Println(dbg)
					//fmt.Println(p)
					p.AppendCmd(cmdtodev)
					runinfo.CurCmd = cmdtodev
					p.SendCmd(d)
					return
				}
			}
			//走到这里说明没有配置串口信息
			log := "Can't find Deivice ID ComInfo pls check config." + runinfo.CurMsg
			mylog.RecordLog(log)
			return
		}
	}
	//走到这里说明没有找到命令的ID设备
	log := "Can't find Deivice ID pls check config or cmd." + runinfo.CurMsg
	mylog.RecordLog(log)
}
func processMatrix(id string, in string, out string, runinfo *dev.RunInfo) {
	for i, _ := range runinfo.Config.Devicedata.Dev {
		d := &runinfo.Config.Devicedata.Dev[i]
		if d.Dtype != dev.MATRIX {
			continue
		}
		if d.Id == id {
			cmd := d.Cmd1
			if len(d.Cmd2) > 0 {
				in = ("000" + in)
				in = in[len(in)-2 : len(in)]
				out = "000" + out
				out = out[len(out)-2 : len(out)]
			}
			cmd = strings.Replace(cmd, "{0}", in, -1)
			cmd = strings.Replace(cmd, "{1}", out, -1)
			for i, _ := range runinfo.DevPeer {
				p := &runinfo.DevPeer[i]
				if p.ConfigInfo.Id == d.Com {
					p.AppendCmd(cmd)
					runinfo.CurCmd = cmd
					p.SendCmd(d)
					return
				}
			}
			//走到这里说明没有配置串口信息
			log := "Can't find Deivice ID ComInfo pls check config." + runinfo.CurMsg
			mylog.RecordLog(log)
			return
		}
	}
	//走到这里说明没有找到命令的ID设备
	log := "Can't find Deivice ID pls check config or cmd." + runinfo.CurMsg
	mylog.RecordLog(log)
}
