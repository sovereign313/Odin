package main

import (
	"os"
	"fmt"
	"net"
	"time"
	"errors"
	"strings"
	"strconv"
	"runtime"
	"syscall"

	"os/exec"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

const authtoken = "Vr6GMEb5IMZjpHezkxvUO0TWLh1ioxbD1"

type Record struct {
	Key string
	Tags map[string]string 
}

type NetDev struct {
	Adapter string
	IPAddress string
}

var cmdbhost string
var tagfile string
var apptagfile string

func GetUUID() string {
	buuid, err := exec.Command("/usr/sbin/dmidecode", "-s", "system-uuid").CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}

	uuid := strings.TrimSpace(string(buuid))
	return string(uuid)
}

func GetHostname() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "", err
	}

	return name, nil
}

func GetMemory() uint64 {
	in := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(in)
	if err != nil {
		return 0
	}

	return uint64(in.Totalram) * uint64(in.Unit)
}

func GetIPs() (map[string]string, error) {
	var netdev map[string]string
	netdev = make(map[string]string) 

	infs, _ := net.Interfaces()
	for _, f := range infs {
		var ip string
		addrs, err := f.Addrs()
		if err != nil {
			fmt.Println(err.Error())
		}

		for _, a := range addrs {
			if a.String() == "" {
				continue
			}
			ip += a.String() + ","
		}

		if len(ip) > 0 {
			ip = ip[:len(ip) - 1]
			netdev["sys." + f.Name] = ip
		}
	}

	return netdev, nil
}

func GetRelease() (string, error) {
	if _, err := os.Stat("/etc/redhat-release"); os.IsNotExist(err) {
		return "generic", nil
        }

	b, err := ioutil.ReadFile("/etc/redhat-release")
	if err != nil {
		return "generic", nil
	}
	return string(b), nil
}

func Register(key string) (bool, error) {
	var tgs map[string]string
	var parms string

	tgs = make(map[string]string)

	if _, err := os.Stat(tagfile); ! os.IsNotExist(err) {
		rawlines, err := ioutil.ReadFile(tagfile)
		if err != nil {
			return false, err
		}

		parts := strings.Split(string(rawlines), "\n")
		for _, line := range parts {
			if line == "" {
				continue
			}

			sep := strings.Split(line, "=")
			k1 := strings.TrimSpace(sep[0])
			v1 := strings.TrimSpace(sep[1])

			if ! strings.Contains(k1, ".") {
				continue
			}

			tgs[k1] = v1
		}
	}

	if _, err := os.Stat(apptagfile); ! os.IsNotExist(err) {
		rawlines, err := ioutil.ReadFile(apptagfile)
		if err != nil {
			return false, err
		}

		parts := strings.Split(string(rawlines), "\n")
		for _, line := range parts {
			if line == "" {
				continue
			}

			sep := strings.Split(line, "=")
			k1 := strings.TrimSpace(sep[0])
			v1 := strings.TrimSpace(sep[1])

			if ! strings.Contains(k1, ".") {
				continue
			}

			tgs[k1] = v1
		}
	}

	hname, errr := GetHostname()
	if errr != nil {
		return false, errr
	}

	netdev, err := GetIPs()
	if err != nil {
		fmt.Println(err.Error())
	}

	now := strconv.Itoa(int(time.Now().Unix()))
	memi := GetMemory()
	mem := strconv.Itoa(int(memi))
	numcpu := runtime.NumCPU()
	cpucount := strconv.Itoa(numcpu)
	release, _ := GetRelease()

	tgs["sys.cpucount"] = cpucount
	tgs["sys.memory"] = mem
	tgs["sys.hostname"] = hname
	tgs["sys.check_in_time"] = now
	tgs["sys.os"] = runtime.GOOS
	tgs["sys.release"] = release 

	for k, v := range netdev {
		tgs[k] = v
	}

	if len(tgs) > 0 {
		tags, err := json.Marshal(tgs)
		if err != nil {
			return false, err
		}

		parms = `token=` + authtoken + `&key=` + key + `&tags=` + string(tags)
	} else {
		parms = `token=` + authtoken + `&key=` + key
	}

	body := strings.NewReader(parms)
	req, err := http.NewRequest("POST", "http://" + cmdbhost + ":8088/register", body)
	if err != nil {
		return false, err 
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	bbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if strings.Contains(string(bbody), "Connection refused") || strings.Contains(string(bbody), "Access Denied") {
		return false, errors.New("Connection Refused")
	}

	if string(bbody) == "Key Already Exists" {
		return false, nil
	}

	if _, err := os.Stat(tagfile); ! os.IsNotExist(err) {
		os.Remove(tagfile)
	}

	if _, err := os.Stat(apptagfile); ! os.IsNotExist(err) {
		os.Remove(apptagfile)
	}

	return true, nil
}

func Update(key string) (bool, error) {
	var tgs map[string]string
	var parms string

	tgs = make(map[string]string)

	if _, err := os.Stat(tagfile); ! os.IsNotExist(err) {
		rawlines, err := ioutil.ReadFile(tagfile)
		if err != nil {
			return false, err
		}

		parts := strings.Split(string(rawlines), "\n")
		for _, line := range parts {
			if line == "" {
				continue
			}

			sep := strings.Split(line, "=")
			k1 := strings.TrimSpace(sep[0])
			v1 := strings.TrimSpace(sep[1])

			if ! strings.Contains(k1, ".") {
				continue
			}

			tgs[k1] = v1
		}

		os.Remove(tagfile)
	}

	if _, err := os.Stat(apptagfile); ! os.IsNotExist(err) {
		rawlines, err := ioutil.ReadFile(apptagfile)
		if err != nil {
			return false, err
		}

		parts := strings.Split(string(rawlines), "\n")
		for _, line := range parts {
			if line == "" {
				continue
			}

			sep := strings.Split(line, "=")
			k1 := strings.TrimSpace(sep[0])
			v1 := strings.TrimSpace(sep[1])

			if ! strings.Contains(k1, ".") {
				continue
			}

			tgs[k1] = v1
		}

		os.Remove(apptagfile)
	}

	hname, errr := GetHostname()
	if errr != nil {
		return false, errr
	}

	netdev, err := GetIPs()
	if err != nil {
		fmt.Println(err.Error())
	}

	now := strconv.Itoa(int(time.Now().Unix()))
	numcpu := runtime.NumCPU()
	cpucount := strconv.Itoa(numcpu)
	memi := GetMemory()
	mem := strconv.Itoa(int(memi))
	release, _ := GetRelease()

	tgs["sys.memory"] = mem
	tgs["sys.cpucount"] = cpucount
	tgs["sys.hostname"] = hname
	tgs["sys.check_in_time"] = now
	tgs["sys.os"] = runtime.GOOS
	tgs["sys.release"] = release
 
	for k, v := range netdev {
		tgs[k] = v
	}

	if len(tgs) > 0 {
		tags, err := json.Marshal(tgs)
		if err != nil {
			return false, err
		}

		parms = `token=` + authtoken + `&key=` + key + `&tags=` + string(tags)
	} else {
		parms = `token=` + authtoken + `&key=` + key
	}

	body := strings.NewReader(parms)
	req, err := http.NewRequest("POST", "http://" + cmdbhost + ":8088/updaterecord", body)
	if err != nil {
		return false, err 
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	bbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if strings.Contains(string(bbody), "Connection refused") || strings.Contains(string(bbody), "Access Denied") {
		return false, errors.New("Connection Refused")
	}

	if string(bbody) == "Key Already Exists" {
		return false, nil
	}

	return true, nil
}

func main() {
	cmdbhost = os.Getenv("cmdbhost")
	if len(cmdbhost) == 0 {
		cmdbhost = "pu1cmdb1001.cac.com"
	}

	tagfile = os.Getenv("tagfile")
	if len(tagfile) == 0 {
		tagfile = "/etc/tags"
	}

	apptagfile = os.Getenv("apptagfile")
	if len(apptagfile) == 0 {
		apptagfile = "/proj/app/tags"
	}

	ok, err := Register(GetUUID())
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if !ok {
		k, errr := Update(GetUUID())
		if errr != nil {
			fmt.Println(err.Error())
			return
		}

		if !k {
			fmt.Println("Something Not Good Took Place")
			return
		}
	}

	for {
		time.Sleep(10 * time.Minute)
		go Update(GetUUID())
	}
}

