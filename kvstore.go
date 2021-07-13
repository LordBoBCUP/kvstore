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
	"github.com/gofrs/uuid"
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
	Key       uuid.UUID `json:"key"`
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

	err = ioutil.WriteFile(storeLocation + "db.db", data, 0777)
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

func ExpireOldKeys() {
	for i, v := range kvstore.DB {
		t1 := v.Date.Add(time.Hour * 1)
		if time.Now().After(t1) {
			// Revoke this user
			fmt.Printf("Removing Key: %v", v)
			kvstore.DB = RemoveIndex(kvstore.DB, i)
		}
	}
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

	if row.Key.String() == "" {
		return errors.New("Key was not completed.")
	}

	if row.Username == "" {
		return errors.New("Username was not completed.")
	}

	removeExistingUser(row.Username)

	kvstore.DB = append(kvstore.DB, *row)

	err := WriteToFile()
	if err != nil {
		return err
	}
	return nil
}

func removeExistingUser(user string) {
	fmt.Println("here")
	for i, v := range kvstore.DB {
		if v.Username == user {
			fmt.Println("Removing item", i)
			kvstore.DB = RemoveIndex(kvstore.DB, i)
		}
	}
}

func ValidateLogin(user string, key string) error {
	ExpireOldKeys()
	for _, v := range kvstore.DB {
		if v.Username == user && v.Key == uuid.FromStringOrNil(key) {
			return nil
		}
	}

	return errors.New("User and key combination not found in database.")
}
