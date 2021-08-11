package plugin

import (
	"FscanX/config"
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	pkt = "\x00" + // session
		"\x00\x00\xc0" + // legth
		"\xfeSMB@\x00" + // protocol
		//[MS-SMB2]: SMB2 NEGOTIATE Request
		//https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-smb2/e14db7ff-763a-4263-8b10-0c3944f52fc5
		"\x00\x00" +
		"\x00\x00" +
		"\x00\x00" +
		"\x00\x00" +
		"\x1f\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		// [MS-SMB2]: SMB2 NEGOTIATE_CONTEXT
		// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-smb2/15332256-522e-4a53-8cd7-0bd17678a2f7
		"$\x00" +
		"\x08\x00" +
		"\x01\x00" +
		"\x00\x00" +
		"\x7f\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"x\x00" +
		"\x00\x00" +
		"\x02\x00" +
		"\x00\x00" +
		"\x02\x02" +
		"\x10\x02" +
		"\x22\x02" +
		"$\x02" +
		"\x00\x03" +
		"\x02\x03" +
		"\x10\x03" +
		"\x11\x03" +
		"\x00\x00\x00\x00" +
		// [MS-SMB2]: SMB2_PREAUTH_INTEGRITY_CAPABILITIES
		// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-smb2/5a07bd66-4734-4af8-abcf-5a44ff7ee0e5
		"\x01\x00" +
		"&\x00" +
		"\x00\x00\x00\x00" +
		"\x01\x00" +
		"\x20\x00" +
		"\x01\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00\x00\x00" +
		"\x00\x00" +
		// [MS-SMB2]: SMB2_COMPRESSION_CAPABILITIES
		// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-smb2/78e0c942-ab41-472b-b117-4a95ebe88271
		"\x03\x00" +
		"\x0e\x00" +
		"\x00\x00\x00\x00" +
		"\x01\x00" + //CompressionAlgorithmCount
		"\x00\x00" +
		"\x01\x00\x00\x00" +
		"\x01\x00" + //LZNT1
		"\x00\x00" +
		"\x00\x00\x00\x00"
)

func SMBGHOST(info *config.HostData){
	err , result := smbghostScan(info)
	if err != nil && !strings.Contains(err.Error(),"timeout"){
		config.WriteLogFile(config.LogFile,fmt.Sprintf("[*] %s",info.HostName),config.Inlog)
	}
	if result != ""{
		config.WriteLogFile(config.LogFile,result,config.Inlog)
	}
}

func smbghostScan(info *config.HostData) (err error,result string) {
	var addr = fmt.Sprintf("%s:%v",info.HostName,445)
	conn, err := net.DialTimeout("tcp",addr,time.Duration(1)*time.Second)
	if err != nil {
		return err,result
	}
	_, err = conn.Write([]byte(pkt))
	if err != nil {
		return err,result
	}
	var buf = make([]byte,1024)
	err = conn.SetReadDeadline(time.Now().Add(time.Duration(1)*time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		return err,result
	}
	defer func(){
		_ = conn.Close()
	}()
	if bytes.Contains(buf[:n],[]byte("Public")){
		result = fmt.Sprintf("[+] %s [CVE-2020-0796]",info.HostName)
	}else{
		result = fmt.Sprintf("[*] %s",info.HostName)
	}
	return nil,result
}