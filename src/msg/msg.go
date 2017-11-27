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

	conn := getUDPListenCon(service)
	if conn == nil {
		return
	}
	runinfo.Cmdconn = conn

	logservice := ":10010"
	conn = getUDPListenCon(logservice)
	if conn == nil {
		return
	}
	runinfo.Logconn = conn

	d := len(runinfo.Config.Devicedata.Dev)
	p := len(runinfo.Config.Comdata.Com)
	log := fmt.Sprintf("Init complete. Total %d devices , %d peers.", d, p)
	mylog.RecordLog(log)
	return
}
func getUDPListenCon(address string) *net.UDPConn {
	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		mylog.RecordLog("err at getUDPListenCon(): " + err.Error() + " " + address)
		return nil
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		mylog.RecordLog("err at getUDPListenCon().ListenUDP: " + err.Error() + " " + address)
		return nil
	}
	return conn
}
func ReciveMsg(conn *net.UDPConn, c chan string) {
	var buf [1024]byte
	for {
		n, rAddr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			log := fmt.Sprintf("Recv msg error: %v ", err.Error())
			mylog.RecordLog(log)
			continue
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
			log := fmt.Sprintf("Recv msg error: %v ", err.Error())
			mylog.RecordPlayLog(log)
			continue
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
			cmds := strings.Split(msg+",0", ",")
			if len(cmds) < 3 {
				log := "error cmd:" + msg
				mylog.RecordLog(log)
				continue
			}
			runinfo.CurMsg = msg
			ProcessCmd(runinfo, cmds[0], cmds[1], cmds[2:]...)
		}
		//fmt.Println(runinfo.DevPeer[0])
		//fmt.Println(runinfo)
	}
}

//args[0]-open, close
//args[0]-IN args[1]-OUT
func ProcessCmd(runinfo *dev.RunInfo, cmdtype string, id string, args ...string) {
	dtype := getDevType(cmdtype)
	if dtype == "" {
		log := "error cmdtype:" + runinfo.CurMsg
		mylog.RecordLog(log)
		return
	}
	for i, _ := range runinfo.Config.Devicedata.Dev {
		d := &runinfo.Config.Devicedata.Dev[i]
		if d.Dtype != dtype {
			continue
		}
		if d.Id == id {
			for i, _ := range runinfo.DevPeer {
				p := &runinfo.DevPeer[i]
				if p.ConfigInfo.Id == d.Com {
					curCmd := d.GetCmdString(args[0], args[1])
					p.AppendCmd(curCmd)
					runinfo.CurCmd = curCmd
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
	return
}

func getDevType(cmdtype string) string {
	//fmt.Println("DEBUGINFO:getDevType()" + cmdtype)
	switch strings.ToLower(cmdtype) {
	case "device_ctrl":
		return dev.SWITCH
	case "matrix_ctrl":
		return dev.MATRIX
	default:
		return ""
	}

}
