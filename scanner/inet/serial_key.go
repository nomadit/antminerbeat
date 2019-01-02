package inet

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var serialKeyFile string
var defaultSerialKey = "123456789*"

func init()  {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	serialKeyFile = dir + "/../serial_key"
}

func readSerialKey() string {
	key := defaultSerialKey
	_, err := os.Stat(serialKeyFile)
	if err != nil {
		if os.IsExist(err) {
			serialKeyBytes := []byte(key)
			err = ioutil.WriteFile(serialKeyFile, serialKeyBytes, 0644)
			if err != nil {
				log.Fatal(err)
			}
			return key
		} else {
			log.Fatal(err)
		}
	} else {
		file, err := os.Open(serialKeyFile)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lines := []string{}
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
		key = lines[0]
	}
	return key
}

func checkNUpdateSerialKey(old string, new string) {
	if old == new {
		return
	}
	_, err := os.Stat(serialKeyFile)
	if err != nil {
		if !os.IsExist(err) {
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	} else {
		err := os.Truncate(serialKeyFile, 0)
		if err != nil {
			log.Fatal(err)
		}
	}
	serialKeyBytes := []byte(new+"\n")
	err = ioutil.WriteFile(serialKeyFile, serialKeyBytes, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
