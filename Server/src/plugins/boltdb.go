package main

import (
	"errors"

	"encoding/json"
	"github.com/boltdb/bolt"
)

type Record struct {
        Key string
        Tags map[string]string
}

var db *bolt.DB
var err error

func ConnectDB() (error) {
	db, err = bolt.Open("/proj/app/cmdb/data/bolt/bolt.db", 0644, nil)
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

        err = db.Update(func(txn *bolt.Tx) error {
		bucket, err := txn.CreateBucketIfNotExists([]byte("odin"))
		if err != nil {
			return err
		}

                err = bucket.Put([]byte(record.Key), jsn)
		if err != nil {
			return err
		}

		return nil
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

        err = db.Update(func(txn *bolt.Tx) error {
		bucket, err := txn.CreateBucketIfNotExists([]byte("odin"))
		if err != nil {
			return err
		}

                err = bucket.Put([]byte(record.Key), jsn)
		if err != nil {
			return err
		}

		return nil
        })

        if err != nil {
                return err
        }

        return nil
}

func GetRecord(key string) (string, error) {
        tmprecord := Record{}
        err := db.View(func(txn *bolt.Tx) error {
		bucket := txn.Bucket([]byte("odin"))
		if bucket == nil {
			return errors.New("Bucket Missing!")
		}

		val := bucket.Get([]byte(key))
		err = json.Unmarshal(val, &tmprecord)
		if err != nil {
			return err
		}

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
        err := db.View(func(txn *bolt.Tx) error {
		bucket := txn.Bucket([]byte("odin"))
		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			tmprecord := Record{}
			err := json.Unmarshal(v, &tmprecord)
			if err != nil {
				continue
			}

			tmprecords = append(tmprecords, tmprecord)
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


var PluginName = "BoltDB"
var PluginVersion = "0.1"

