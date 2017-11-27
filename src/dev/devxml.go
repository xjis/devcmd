package dev

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"mylog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RunInfo struct {
	Cmdconn *net.UDPConn
	Logconn *net.UDPConn
	CmdQ    chan string
	CurMsg  string //当前处理命令源
	CurCmd  string //当前发送给设备的命令字
	DevPeer []DevicePeerInfo
	Config  *Config
}
type DevicePeerInfo struct {
	Com        *Serial
	Conn       net.Conn
	ConnSta    int //1正常  0 未连接
	CmdBuff    string
	ConfigInfo *ComInfo
	m          *sync.Mutex
}
type Serial struct {
	BaudRate int
	DataBits int
	StopBits int
	Parity   int
}
type Config struct {
	XMLName    xml.Name `xml:"AppConfig"`
	Serverdata Servers  `xml:"ServerInfo"`
	Comdata    Coms     `xml:"ComInfo"`
	Devicedata Devices  `xml:"DeviceList"`
}
type Servers struct {
	XMLName xml.Name     `xml:"ServerInfo"`
	Server  []ServerInfo `xml:"Server"`
}
type ServerInfo struct {
	XMLName xml.Name `xml:"Server"`
	Ip      string   `xml:"ip,attr"`
	Port    string   `xml:"port,attr"`
}
type Coms struct {
	XMLName xml.Name  `xml:"ComInfo"`
	Com     []ComInfo `xml:"com"`
}
type ComInfo struct {
	XMLName  xml.Name `xml:"com"`
	Id       string   `xml:"id,attr"`
	BaudRate string   `xml:"BaudRate,attr"`
	DataBits string   `xml:"DataBits,attr"`
	StopBits string   `xml:"StopBits,attr"`
	Parity   string   `xml:"Parity,attr"`
	Cmdstr   string   `xml:"cmdstr,attr"`
	Ip       string   `xml:"ip,attr"`
	Port     string   `xml:"port,attr"`
	Protocol string   `xml:"protocol,attr"`
}
type Devices struct {
	XMLName xml.Name     `xml:"DeviceList"`
	Dev     []DeviceInfo `xml:"device"`
}
type DeviceInfo struct {
	XMLName    xml.Name `xml:"device"`
	Dtype      string   `xml:"type,attr"`
	Id         string   `xml:"id,attr"`
	Id2        string   `xml:"id2,attr"`
	Id3        string   `xml:"id3,attr"`
	Devicename string   `xml:"devicename"`
	Com        string   `xml:"com"`
	Cmd1       string   `xml:"cmd1"`
	Cmd2       string   `xml:"cmd2"`
	Cmd3       string   `xml:"cmd3"`
	Cmd4       string   `xml:"cmd4"`
}

const (
	SWITCH = "1"
	MATRIX = "2"
)

func ReadConfigFile(FileName string) *Config {

	file, err := os.Open(FileName) // For read access.
	if err != nil {
		s := fmt.Sprintf("ReadConfigFile error at open: %v", err)
		mylog.RecordLog(s)
		return nil
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		s := fmt.Sprintf("ReadConfigFile error at readall: %v", err)
		mylog.RecordLog(s)
		return nil
	}
	v := Config{}
	err = xml.Unmarshal(data, &v)
	if err != nil {
		s := fmt.Sprintf("ReadConfigFile error at Unmarshal: %v", err)
		mylog.RecordLog(s)
		return nil
	}

	return &v
}

//初始化对端设备信息
func (runinfo *RunInfo) InitPeer() int {
	if runinfo == nil {
		return 0
	}
	count := len(runinfo.Config.Comdata.Com)
	runinfo.DevPeer = make([]DevicePeerInfo, count)
	for i := 0; i < count; i++ {
		runinfo.DevPeer[i].ConfigInfo = &runinfo.Config.Comdata.Com[i]
		runinfo.DevPeer[i].CmdBuff = ""
		runinfo.DevPeer[i].ConnSta = 0
		runinfo.DevPeer[i].m = new(sync.Mutex)
		go (&runinfo.DevPeer[i]).ConnectToPeer(nil) //初始化的时候尝试连接对方
	}
	return 0
}
func (p *DevicePeerInfo) SendCmd(to *DeviceInfo) {
	if p.ConnSta == 0 {
		go p.ConnectToPeer(to)
		return
	}
	if p.ConfigInfo.Id == "0" {
		sendStartPCPacket(p, to)
		p.ClearCmd()
		return
	}
	if len(p.CmdBuff) < 1 {
		return
	}
	b := []byte(p.CmdBuff)          //字符串格式
	if p.ConfigInfo.Cmdstr == "0" { //二进制格式
		b = HexToBye(p.CmdBuff)
	}
	p.Conn.Write(b)
	time.Sleep(100 * time.Millisecond)
	_, err := p.Conn.Write(b)
	if err != nil {
		p.m.Lock()
		p.ConnSta = 0
		p.m.Unlock()
		mylog.RecordLog(err.Error())
		return
	}
	//dbg := "DBGINFO:SendCmd() p.CmdBuff:" + p.CmdBuff
	//fmt.Println(dbg)
	log := "Send to " + to.Devicename + " Cmd:" + p.CmdBuff
	mylog.RecordLog(log)
	p.ClearCmd()
	return
}
func sendStartPCPacket(p *DevicePeerInfo, to *DeviceInfo) {
	macs := strings.Split(p.CmdBuff, ",")
	for _, mac := range macs {
		if mac == "" {
			continue
		}
		if len(mac) < 12 {
			log := "发送开机报文长度错误 mac：" + mac
			mylog.RecordLog(log)
		}
		mac = strings.Replace(mac, "-", "", -1)
		pkt := "FFFFFFFFFFFF" + strings.Repeat(mac, 16)
		b := HexToBye(pkt)
		p.Conn.Write(b)
		_, err := p.Conn.Write(b)
		if err != nil {
			p.m.Lock()
			p.ConnSta = 0
			p.m.Unlock()
			mylog.RecordLog(err.Error())
			return
		}
		log := "Send to start PC packet mac:" + mac
		mylog.RecordLog(log)
	}
	return
}

//连接函数
func (p *DevicePeerInfo) ConnectToPeer(to *DeviceInfo) {
	if p.ConfigInfo.Protocol == "tcp" {
		service := p.ConfigInfo.Ip + ":" + p.ConfigInfo.Port
		tcpAddr, err := net.ResolveTCPAddr("tcp", service)
		logError(err)
		if err != nil {
			return
		}
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		logError(err)
		if err != nil {
			return
		}
		p.Conn = conn
		p.m.Lock()
		p.ConnSta = 1
		p.m.Unlock()
		if to != nil {
			p.SendCmd(to)
		}
	}
	if p.ConfigInfo.Protocol == "udp" {
		service := p.ConfigInfo.Ip + ":" + p.ConfigInfo.Port
		conn, err := net.Dial("udp4", service)
		logError(err)
		if err != nil {
			return
		}
		p.Conn = conn
		p.m.Lock()
		p.ConnSta = 1
		p.m.Unlock()
		if to != nil {
			p.SendCmd(to)
		}
	}
}

func (p *DevicePeerInfo) AppendCmd(cmd string) {
	p.m.Lock()
	p.CmdBuff += cmd
	p.m.Unlock()
}

func (p *DevicePeerInfo) ClearCmd() {
	p.m.Lock()
	p.CmdBuff = ""
	p.m.Unlock()
}

//arg-未使用
func (d *DeviceInfo) GetCmdString(cmd1 string, cmd2 string, args ...string) string {
	if d.Dtype == SWITCH {
		cmd := d.Cmd1
		if strings.ToLower(cmd1) == "close" {
			cmd = d.Cmd2
		}
		return cmd
	}
	if d.Dtype == MATRIX {
		cmd := d.Cmd1
		in := cmd1
		out := cmd2
		if len(d.Cmd2) > 0 {
			in = ("000" + in)
			in = in[len(in)-2 : len(in)]
			out = "000" + out
			out = out[len(out)-2 : len(out)]
		}
		cmd = strings.Replace(cmd, "{0}", in, -1)
		cmd = strings.Replace(cmd, "{1}", out, -1)
		return cmd
	}
	return ""
}

//16进制字符串转[]byte
func HexToBye(hex string) []byte {
	length := len(hex) / 2
	slice := make([]byte, length)
	rs := []rune(hex)

	for i := 0; i < length; i++ {
		s := string(rs[i*2 : i*2+2])
		value, _ := strconv.ParseInt(s, 16, 10)
		slice[i] = byte(value & 0xFF)
	}
	return slice
}

func logError(err error) {
	if err != nil {
		mylog.RecordLog(err.Error())
	}
}
