package xbvr

import (
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"time"
	
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func GetDB() (*gorm.DB, error) {
	db, err := gorm.Open("sqlite3", filepath.Join(appDir, "main.db"))
	if err != nil {
		log.Fatal("failed to connect database", err)
	}
	return db, nil
}

func waitForServices(services []string, timeOut time.Duration) error {
	var depChan = make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(len(services))
	go func() {
		for _, s := range services {
			go func(s string) {
				defer wg.Done()
				for {
					_, err := net.Dial("tcp", s)
					if err == nil {
						return
					}
					time.Sleep(100 * time.Millisecond)
				}
			}(s)
		}
		wg.Wait()
		close(depChan)
	}()

	select {
	case <-depChan: // services are ready
		return nil
	case <-time.After(timeOut):
		return fmt.Errorf("services aren't ready in %s", timeOut)
	}
}

func init() {
	initPaths()

	db, _ := GetDB()
	defer db.Close()

	db.AutoMigrate(&Scene{})
	db.AutoMigrate(&Actor{})
	db.AutoMigrate(&Tag{})
	db.AutoMigrate(&Image{})

	db.AutoMigrate(&File{})
	db.AutoMigrate(&Volume{})

	db.AutoMigrate(&KV{})
}
