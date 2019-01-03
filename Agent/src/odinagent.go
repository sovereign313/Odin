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

	"net/http"
	"io/ioutil"
	"encoding/json"

	"github.com/ochapman/godmi"
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

func GetUUID() string {
	godmi.Init()
	sysinfo := godmi.GetSystemInformation()

	return sysinfo.UUID
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
			netdev[f.Name] = ip
		}
	}

	return netdev, nil
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
			sep := strings.Split(line, "=")
			k1 := strings.TrimSpace(sep[0])
			v1 := strings.TrimSpace(sep[1])
			tgs[k1] = v1
		}

		os.Remove(tagfile)
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

	tgs["cpucount"] = cpucount
	tgs["memory"] = mem
	tgs["hostname"] = hname
	tgs["check_in_time"] = now
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
			sep := strings.Split(line, "=")
			k1 := strings.TrimSpace(sep[0])
			v1 := strings.TrimSpace(sep[1])
			tgs[k1] = v1
		}

		os.Remove(tagfile)
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

	tgs["memory"] = mem
	tgs["cpucount"] = cpucount
	tgs["hostname"] = hname
	tgs["check_in_time"] = now
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

