package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"runtime"
	"sync"
	"time"

	"github.com/bradfitz/slice"
)

var (
	kvstore       *DB
	once          sync.Once
	storeLocation string
)

type DB struct {
	DB []Row `json:"DB"`
}

type Row struct {
	Id        int       `json:"id"`
	Username  string    `json:"username"`
	Key       string    `json:"key"`
	Date      time.Time `json:"date"`
	IpAddress string    `json:"ip_address"`
}

func GetDB(location string) *DB {
	once.Do(func() {
		if runtime.GOOS == "windows" {
			if string(location[len(location)-1:]) != "\\" {
				storeLocation = location + "\\"
			}
			storeLocation = location
		}
		if runtime.GOOS == "linux" {
			if string(location[len(location)-1:]) != "/" {
				storeLocation = location + "/"
			}
			storeLocation = location
		}

		data, err := ReadFromFile()
		if err != nil {
			fmt.Println(err)
			panic("Unable to read kvstore db.db - Application exiting.")
		}
		kvstore = data
	})
	return kvstore
}

func getNextID() int {
	if len(kvstore.DB) < 1 {
		return 1
	}

	slice.Sort(kvstore.DB[:], func(i, j int) bool {
		return kvstore.DB[i].Id < kvstore.DB[j].Id
	})

	v := kvstore.DB[len(kvstore.DB)-1]
	return v.Id + 1
}

func WriteToFile() error {
	data, err := json.Marshal(kvstore)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(storeLocation+"db.db", data, 0777)
	return err

}

func ReadFromFile() (*DB, error) {
	data, err := ioutil.ReadFile(storeLocation + "db.db")
	if err != nil {
		return nil, err
	}

	db := DB{}

	err = json.Unmarshal(data, &db)
	return &db, err
}

func ExpireOldKeys() error {
	tmp := kvstore.DB[:0]

	for _, v := range kvstore.DB {
		t1 := v.Date.Add(time.Hour * 8)
		if time.Now().Before(t1) {
			tmp = append(tmp, v)
		}
	}
	kvstore.DB = tmp
	err := WriteToFile()
	return err
}

func RemoveIndex(s []Row, index int) []Row {
	return append(s[:index], s[index+1:]...)
}

func AddRow(row *Row) error {
	row.Id = getNextID()
	row.Date = time.Now()

	if row.IpAddress == "" {
		return errors.New("IP Address was not completed")
	}

	if row.Key == "" {
		return errors.New("Key was not completed.")
	}

	if row.Username == "" {
		return errors.New("Username was not completed.")
	}

	err := removeExistingUser(row.Username)
	if err != nil {
		return err
	}

	kvstore.DB = append(kvstore.DB, *row)

	err = WriteToFile()
	if err != nil {
		return err
	}
	return nil
}

func removeExistingUser(user string) error {
	fmt.Println("here")
	for i, v := range kvstore.DB {
		if v.Username == user {
			fmt.Println("Removing item", i)
			kvstore.DB = RemoveIndex(kvstore.DB, i)
		}
	}

	err := WriteToFile()
	return err
}

func ValidateLogin(user string, key string, ip string) error {
	err := ExpireOldKeys()
	if err != nil {
		return err
	}

	for _, v := range kvstore.DB {
		if v.Username == user && v.Key == key {
			if v.IpAddress == ip {
				return nil
			}
			return errors.New("Username and password accepted, IP address didn't match KeyStore object.")
		}
	}

	return errors.New("User and key combination not found in database.")
}

func GetExistingUser(user string) (string, error) {
	var res string

	// Expire any old keys first.
	ExpireOldKeys()

	for i, v := range kvstore.DB {
		if v.Username == user {
			fmt.Println("Returning item", i)
			json, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			res = string(json)
		}
	}

	return res, nil
}
