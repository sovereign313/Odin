package main

import (
	"os"
	"fmt"
	"time"
	"errors"
	"strings"

	"io/ioutil"
        "math/rand"
	"net/http"
)

const authtoken = "Vr6GMEb5IMZjpHezkxvUO0TWLh1ioxbD1"

var cmdbhost string

func main() {
	cmdbhost = os.Getenv("cmdbhost")
	if len(cmdbhost) == 0 {
		cmdbhost = "pu1cmdb1001.cac.com"
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: " + os.Args[0] + " <team>")
		fmt.Println("ie: " + os.Args[0] + " mw  # For Middleware")
		return
	}

	apikey := RandomString(12)
	err := UpdateCMDB(apikey, os.Args[1])
	if err != nil {
		fmt.Println("Failed To Generate Key: " + err.Error())
		return
	}

	msg := "API Key: " + apikey + "\n"
	msg += "Team: " + os.Args[1] + "\n"
	msg += "URL: http://" + cmdbhost + ":8088\n"
	fmt.Println(msg)
}

func UpdateCMDB(key string, org string) error {
	body := strings.NewReader(`token=` + authtoken + `&apikey=` + key + `&apivalue=` + org) 
	req, err := http.NewRequest("POST", "http://" + cmdbhost + ":8088/updateapikeys", body) 
	if err != nil { 
		return err
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded") 
	resp, err := http.DefaultClient.Do(req) 
	if err != nil { 
		return err	
	} 
	defer resp.Body.Close()

        bbody, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return err
        }

        if strings.Contains(string(bbody), "Connection refused") || strings.Contains(string(bbody), "Access Denied") {
                return errors.New(string(bbody))
        }

        if strings.Contains(string(bbody), "API Key Is Already In Use") {
                return errors.New(string(bbody))
        }

	return nil
}

func RandomString(length int) string {

        rand.Seed(time.Now().UnixNano())

        var list = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

        chars := make([]rune, length)
        for i := range chars {
                chars[i] = list[rand.Intn(len(list))]
        }

        return string(chars)
}

