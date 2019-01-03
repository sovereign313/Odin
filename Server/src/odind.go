package main

import (
	"fmt"

	"net/http"
	"encoding/json"

        "github.com/gorilla/mux"
	"github.com/dgraph-io/badger"
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

var db *badger.DB

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

	if len(tags) != 0 {
		err := json.Unmarshal([]byte(tags), &tmptags)
		if err != nil {
			fmt.Fprintf(w, "Error Parsing Tags: " + err.Error())
			return
		}
	}

	flag := false
	rec, err := GetRecord(db, key)
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
	tmpRecord.Key = key 
	if len(tmptags) != 0 {
		tmpRecord.Tags = tmptags 
	}

	err = InsertRecord(db, tmpRecord)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	fmt.Fprintf(w, "Success\n")
	return	
}

func handleUpdateRecord(w http.ResponseWriter, r *http.Request) {
	var tmptags map[string]string
	key := r.FormValue("key")
	tags := r.FormValue("tags")

	err := json.Unmarshal([]byte(tags), &tmptags)
	if err != nil {
		fmt.Fprintf(w, "Error Parsing Tags: " + err.Error())
		return
	}

	tmpRecord, err := GetRecord(db, key)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	for k, v := range tmptags {
		tmpRecord.Tags[k] = v
	}
	
	err = UpdateRecord(db, tmpRecord)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	fmt.Fprintf(w, "Success\n")
	return	
}

func handleGetRecord(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")

	rec, err := GetRecord(db, key)
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

	rec, err := GetRecords(db)
	if err != nil {
		fmt.Println(err.Error())
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
			if k == tag || v == tag {
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

func main() {
	var err error

        router := mux.NewRouter()
        router.HandleFunc("/whoareyou", handleWhoAreYou)
        router.HandleFunc("/ping", handlePing)
        router.HandleFunc("/description", handleDescription)
	router.HandleFunc("/register", handleRegister)
	router.HandleFunc("/getrecords", handleGetRecords)
	router.HandleFunc("/getrecord", handleGetRecord)
	router.HandleFunc("/updaterecord", handleUpdateRecord)
        router.HandleFunc("/", handleHelp)

	db, err = ConnectDB()
	if err != nil {
		fmt.Println(err.Error())
		return
	}


        err = http.ListenAndServe(":8088", router)
        if err != nil {
                fmt.Println("ListenAndServe: ", err)
        }
}

func UpdateRecord(db *badger.DB, record Record) error {
	jsn, err := json.Marshal(record)
	if err != nil {
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(record.Key), jsn)
		return err
	})

	if err != nil {
		return err
	}

	return nil
}

func GetRecord(db *badger.DB, key string) (Record, error) {
	tmprecord := Record{}
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			err = json.Unmarshal(val, &tmprecord)
			if err != nil {
				return err
			}

			return nil
		})
		
		return nil
	})

	if err != nil {
		return tmprecord, err
	}

	return tmprecord, nil
}

func GetRecords(db *badger.DB) ([]Record, error) {
	tmprecords := []Record{}
	tmprecord := Record{}
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				err := json.Unmarshal(v, &tmprecord)
				if err != nil {
					return err
				}

				tmprecords = append(tmprecords, tmprecord)
				return nil
			})

			if err != nil {
				return err	
			}
		}

		return nil
	})

	if err != nil {
		return tmprecords, err
	}

	return tmprecords, nil
}

func InsertRecord(db *badger.DB, record Record) error {
	jsn, err := json.Marshal(record)
	if err != nil {
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(record.Key), jsn)
		return err
	})

	if err != nil {
		return err
	}

	return nil
}

func ConnectDB() (*badger.DB, error) {
	opts := badger.DefaultOptions
	opts.Dir = "/proj/app/cmdb"
	opts.ValueDir = "/proj/app/cmdb"
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	
	return db, nil
}

func CloseDB(db *badger.DB) {
	db.Close()
}
