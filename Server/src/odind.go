package main

import (
	"os"
	"fmt"
	"time"
	"bufio"
	"errors"
	"plugin"
	"strings"
	"strconv"

	"net/http"
	"path/filepath"
	"encoding/json"

        "github.com/gorilla/mux"
	"github.com/gorilla/handlers"
)

const (
	dbname = "cmdbcollection"
	collectionname = "servers"
	authtoken = "Vr6GMEb5IMZjpHezkxvUO0TWLh1ioxbD1"
)

type Record struct {
	Key string
	Tags map[string]string 
}

type PluginList struct {
        Name string
        Version string
	ConnectDB func() (error) 
	CloseDB func() (error)
	InsertRecord func(string) (error)
	UpdateRecord func(string) (error)
	GetRecord func(string) (string, error)
	GetRecords func() (string, error)
}

var dcs []string
var apikeys map[string]string
var plugins []PluginList

func handleWhoAreYou(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Odin CMDB Server")
}

func handlePing(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "pong")
}

func handleDescription(w http.ResponseWriter, r *http.Request) {
        html := "Server Side Application For Odin CMDB"
        fmt.Fprintf(w, html)
}

func handleHelp(w http.ResponseWriter, r *http.Request) {
	html := "Odin CMDB Help"
	fmt.Fprintf(w, html)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var tmptags map[string]string
        token := r.FormValue("token")
	key := r.FormValue("key")
	tags := r.FormValue("tags")

	if token != authtoken {
		fmt.Fprintf(w, "Unauthorized Attempt To Register")
		return
	}

	if len(key) == 0 {
		fmt.Fprintf(w, "Missing 'key' parameter")
		return
	}

	if len(tags) != 0 {
		err := json.Unmarshal([]byte(tags), &tmptags)
		if err != nil {
			fmt.Fprintf(w, "Error Parsing Tags: " + err.Error())
			return
		}
	}

	flag := false
	rec, err := GetRecord(key)
	if err != nil {
		if err.Error() == "Key not found" {
			flag = true
		}
	}

	if rec.Key == key {
		fmt.Fprintf(w, "Key Already Exists")
		return
	}

	if ! flag {
		fmt.Fprintf(w, err.Error())
		return
	}

	tmpRecord := Record{}
	tmpRecord.Key = strings.TrimSpace(key)
	if len(tmptags) != 0 {
		tmpRecord.Tags = tmptags 
	}


        now := strconv.Itoa(int(time.Now().Unix()))
	tmpRecord.Tags["update_time"] = now 
	tmpRecord.Tags["registered_time"] = now

	err = InsertRecord(tmpRecord)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	fmt.Fprintf(w, "Success\n")
	return	
}

func handleUpdateRecord(w http.ResponseWriter, r *http.Request) {
	var tmptags map[string]string

	apikey := r.FormValue("apikey")
	key := r.FormValue("key")
	tags := r.FormValue("tags")
	token := r.FormValue("token")

	if len(key) == 0 {
		fmt.Fprintf(w, "Missing 'key' parameter")
		return
	}

	if len(tags) == 0 {
		fmt.Fprintf(w, "Missing 'tags' parameter")
		return
	}

	if token != authtoken {
		if len(apikey) == 0 {
			fmt.Fprintf(w, "Missing 'apikey' parameter")
			return
		}

		if _, ok := apikeys[apikey]; ! ok {
			fmt.Fprintf(w, "API Key is not authorized or valid")
			return
		}

		err := json.Unmarshal([]byte(tags), &tmptags)
		if err != nil {
			fmt.Fprintf(w, "Error Parsing Tags: " + err.Error())
			return
		}

		flag := false
		for k, _ := range tmptags {
			if ! strings.Contains(k, ".") {
				flag = true
				break
			}

			parts := strings.Split(k, ".")
			if parts[0] != apikeys[apikey] {
				flag = true
				break
			}
		}

		if flag {
			fmt.Fprintf(w, "Failed! Trying To Add/Modify A Key Not Belonging To Your Team")
			return
		}
	} else {
		err := json.Unmarshal([]byte(tags), &tmptags)
		if err != nil {
			fmt.Fprintf(w, "Error Parsing Tags: " + err.Error())
			return
		}
	}

	tmpRecord, err := GetRecord(key)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	flag := false
	for k, v := range tmptags {
		if k == "registered_time" {
			flag = true
		}

		tmpRecord.Tags[k] = v
	}

	if flag {
		fmt.Fprintf(w, "Error: Tried to write to a read only value: registered_time")
		return
	}

        now := strconv.Itoa(int(time.Now().Unix()))
	tmpRecord.Tags["update_time"] = now 
	err = UpdateRecord(tmpRecord)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	fmt.Fprintf(w, "Success\n")
	return	
}

func handleGetRecord(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")

	rec, err := GetRecord(key)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	jsn, err := json.Marshal(rec)
	if err != nil {
		fmt.Fprintf(w, "Failed To Marshal Records Into Json")
		return
	}
	fmt.Fprintf(w, string(jsn))
	return
}

func handleGetRecords(w http.ResponseWriter, r *http.Request) {
	tag := r.FormValue("tag")
	records := []Record{}

	rec, err := GetRecords()
	if err != nil {
		Log(err.Error())
	}

	if len(tag) == 0 {
		jsn, err := json.Marshal(rec)
		if err != nil {
			fmt.Fprintf(w, "Failed To Marshal Records Into Json")
			return
		}
		fmt.Fprintf(w, string(jsn))
		return
	} 

	for _, r := range rec {
		for k, v := range r.Tags {
			if strings.Contains(k, tag) || strings.Contains(v, tag) {
				records = append(records, r)
			}
		}
	}

	jsn, err := json.Marshal(records)
	if err != nil {
		fmt.Fprintf(w, "Failed To Marshal Records Into Json")
		return
	}

	fmt.Fprintf(w, string(jsn))
	return	
}

func handleUpdateAPIKeys(w http.ResponseWriter, r *http.Request) {
        token := r.FormValue("token")
	apikey := r.FormValue("apikey")
	apivalue := r.FormValue("apivalue")

	if token != authtoken {
		fmt.Fprintf(w, "Unauthorized Attempt To Update API Keys")
		return
	}

	if len(apikey) == 0 {
		fmt.Fprintf(w, "Missing 'apikey' Parameter")
		return
	}

	if len(apivalue) == 0 {
		fmt.Fprintf(w, "Missing 'apivalue' Parameter")
		return
	}

	if val, ok := apikeys[apikey]; ok {
		fmt.Fprintf(w, "API Key Is Already In Use: " + val)
		return
	}

	apikeys[apikey] = apivalue

	err := DumpAPIKeys()
	if err != nil {
		fmt.Fprintf(w, "Error Writing API Key File: " + err.Error())
		return
	}

	fmt.Fprintf(w, "success")
}


func main() {
	var err error

	apikeys = make(map[string]string)

	file, err := os.Open("/proj/app/cmdb/cmdbapi.dat")
	if err != nil {
		Log(err.Error())
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "=")
		apikeys[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	if err = scanner.Err(); err != nil {
		Log(err.Error())
		return
	}

        router := mux.NewRouter()
        router.HandleFunc("/whoareyou", handleWhoAreYou)
        router.HandleFunc("/ping", handlePing)
        router.HandleFunc("/description", handleDescription)
	router.HandleFunc("/register", handleRegister)
	router.HandleFunc("/getrecords", handleGetRecords)
	router.HandleFunc("/getrecord", handleGetRecord)
	router.HandleFunc("/updaterecord", handleUpdateRecord)
	router.HandleFunc("/updateapikeys", handleUpdateAPIKeys)
        router.HandleFunc("/", handleHelp)

	err = LoadPlugins("/proj/app/cmdb/cmdbplugins")
	if err != nil {
		Log(err.Error())
		return
	}

	err = ConnectDB()
	if err != nil {
		Log(err.Error())
		return
	}


	corsObj := handlers.AllowedOrigins([]string{"*"})
        err = http.ListenAndServe(":8088", handlers.CORS(corsObj)(router))
        if err != nil {
                Log("ListenAndServe: " + err.Error())
        }
}

func LoadPlugins(plgpath string) error {
	_, er := os.Stat(plgpath)
	if os.IsNotExist(er) {
		Log("Plugin Path Doesn't Exist (" + plgpath + ")")
		Log("No Plugins Loaded")
		return er
	}

        all_plugins, err := filepath.Glob(plgpath + "/*.so")
        if err != nil {
		Log("Error Getting Files From: " + plgpath + ": " + err.Error())
		return err
        }

        for _, filename := range all_plugins {
                p, err := plugin.Open(filename)
                if err != nil {
			return err
                }

                connectsymbol, err := p.Lookup("ConnectDB")
		if err != nil {
			Log("failed to look up ConnectDB: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
		}

                closesymbol, err := p.Lookup("CloseDB")
		if err != nil {
			Log("failed to look up CloseDB: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
		}

                insertsymbol, err := p.Lookup("InsertRecord")
		if err != nil {
			Log("failed to look up InsertRecord: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
		}

                updatesymbol, err := p.Lookup("UpdateRecord")
		if err != nil {
			Log("failed to look up UpdateRecord: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
		}

		getrecordsymbol, err := p.Lookup("GetRecord")
		if err != nil {
			Log("failed to look up GetRecord: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
		}

		getrecordssymbol, err := p.Lookup("GetRecords")
		if err != nil {
			Log("failed to look up GetRecords: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
		}

                nsymbol, err := p.Lookup("PluginName")
		if err != nil {
			Log("failed to look up Plugin Name: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
		}

                vsymbol, err := p.Lookup("PluginVersion")
                if err != nil {
			Log("failed to look up Plugin Version: " + err.Error())
			Log("Plugin Not Loaded: " + filename)
			continue
                }

                plgname, ok := nsymbol.(*string)
		if !ok {
			Log("failed to load name symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
		}

                plgversion, ok := vsymbol.(*string)
		if !ok {
			Log("failed to load version symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
		}

                plgconnect, ok := connectsymbol.(func() (error))
                if !ok {
			Log("failed to load connect symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
                }

                plgclose, ok := closesymbol.(func() (error))
                if !ok {
			Log("failed to load close symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
                }

                plginsert, ok := insertsymbol.(func(string) (error))
                if !ok {
			Log("failed to load insert symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
                }

		plgupdate, ok := updatesymbol.(func(string) (error))
		if !ok {
			Log("failed to load update symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
		}

		plggetrecord, ok := getrecordsymbol.(func(string) (string, error))
		if !ok {
			Log("failed to load getrecord symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
		}

		plggetrecords, ok := getrecordssymbol.(func() (string, error))
		if !ok {
			Log("failed to load getrecords symbol from: " + filename)
			Log("Plugin Not Loaded")
			continue
		}

                tmpplg := PluginList{}
                tmpplg.Name = *plgname
                tmpplg.Version = *plgversion
                tmpplg.ConnectDB = plgconnect
		tmpplg.CloseDB = plgclose
		tmpplg.InsertRecord = plginsert
		tmpplg.UpdateRecord = plgupdate
		tmpplg.GetRecord = plggetrecord
		tmpplg.GetRecords = plggetrecords

		flag := false
		for _, p := range plugins {
			if p.Name == tmpplg.Name {
				Log("Plugin Already Loaded: " + p.Name)
				flag = true
			}
		}

		if ! flag {
	                plugins = append(plugins, tmpplg)
		}
        }

	if len(plugins) < 1 {
		Log("No Plugins Loaded: Do .so files exist in: " + plgpath + "?")
		return errors.New("No Plugins Loaded")
	} else {
		return nil
	}
}

func Log(message string) {
	service := "Odin CMDB"
	loglevel := "INFO"
	
	file, err := os.OpenFile("/proj/app/cmdb/logs/odind.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 
	}
	defer file.Close()

	current_time := time.Now().Local()
	t := current_time.Format("Jan 02 2006 03:04:05")
	_, err = file.WriteString(loglevel + " | " + t + " | " + service + " | " + message + "\n")

	if err != nil {
		return 
	}

	return 
}

func DumpAPIKeys() error {
	f, err := os.Create("/proj/app/cmdb/cmdbapi.dat")
	if err != nil {
		return err
	}
	defer f.Close()

	for k, v := range apikeys {
		f.WriteString(k + " = " + v + "\n")
	}

	f.Sync()
	return nil
}

func UpdateRecord(record Record) error {
	jsn, err := json.Marshal(record)
	if err != nil {
		return err
	}

	for _, p := range plugins {
		err := p.UpdateRecord(string(jsn))
		if err != nil {
			Log("Failed To Update ("+ p.Name + "): " + err.Error())
		} else {
			Log("Successfully Updated (" + p.Name + "): ")
		}
	}

	return nil
}

func GetRecord(key string) (Record, error) {
	tmprecord := Record{}
	p := plugins[0]
	strrecord, err := p.GetRecord(key)
	if err != nil {
		return tmprecord, err
	}

	err = json.Unmarshal([]byte(strrecord), &tmprecord)
	if err != nil {
		return tmprecord, err
	}

	return tmprecord, nil
}

func GetRecords() ([]Record, error) {
	tmprecords := []Record{}
	p := plugins[0]
	strrecords, err := p.GetRecords()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(strrecords), &tmprecords)
	if err != nil {
		return nil, err
	}

	return tmprecords, nil
}

func InsertRecord(record Record) error {
	jsn, err := json.Marshal(record)
	if err != nil {
		return err
	}

	for _, p := range plugins {
		err := p.InsertRecord(string(jsn))
		if err != nil {
			Log("Failed To Insert ("+ p.Name + "): " + err.Error())
		} else {
			Log("Successfully Inserted (" + p.Name + "): ")
		}
	}

	return nil
}

func ConnectDB() (error) {
	for _, p := range plugins {
		err := p.ConnectDB()
		if err != nil {
			Log("Failed To Open ("+ p.Name + "): " + err.Error())
		} else {
			Log("Successfully Opened (" + p.Name + "): ")
		}
	}

	return nil
}

func CloseDB() (error) {
	for _, p := range plugins {
		err := p.CloseDB()
		if err != nil {
			Log("Failed To Close ("+ p.Name + "): " + err.Error())
		} else {
			Log("Successfully Closed (" + p.Name + ")")
		}
	}

	return nil
}
