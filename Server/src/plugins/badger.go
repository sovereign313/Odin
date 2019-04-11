package main

import (
	"encoding/json"

        "github.com/dgraph-io/badger"
)

type Record struct {
        Key string
        Tags map[string]string
}

var db *badger.DB
var err error

func ConnectDB() (error) {
        opts := badger.DefaultOptions
        opts.Dir = "/proj/app/cmdb/data/badger"
        opts.ValueDir = "/proj/app/cmdb/data/badger"
        db, err = badger.Open(opts)
        if err != nil {
                return err
        }

        return nil
}

func CloseDB() (error) {
        db.Close()
	return nil
}

func InsertRecord(strrecord string) error {
	record := Record{}
	err := json.Unmarshal([]byte(strrecord), &record)
	if err != nil {
		return err
	}

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

func UpdateRecord(strrecord string) error {
	record := Record{}
	err := json.Unmarshal([]byte(strrecord), &record)
	if err != nil {
		return err
	}

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

func GetRecord(key string) (string, error) {
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
                return "", err
        }

	jsn, err := json.Marshal(tmprecord)
	if err != nil {
		return "", err
	}

        return string(jsn), nil
}

func GetRecords() (string, error) {
        tmprecords := []Record{}
        err := db.View(func(txn *badger.Txn) error {
                opts := badger.DefaultIteratorOptions
                opts.PrefetchSize = 10
                it := txn.NewIterator(opts)
                defer it.Close()
                for it.Rewind(); it.Valid(); it.Next() {
                        tmprecord := Record{}
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
                return "", err
        }

	jsn, err := json.Marshal(tmprecords)
	if err != nil {
		return "", err
	}

        return string(jsn), nil
}


var PluginName = "BadgerDB"
var PluginVersion = "0.1"

