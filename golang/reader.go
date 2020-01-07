package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	badger "github.com/dgraph-io/badger"
	"github.com/karrick/godirwalk"
)

func countDockerfiles(dirname string) (int, int, error) {
	count := 0
	errors := 0
	err := godirwalk.Walk(dirname, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			log.Printf("%s %s\n", de.ModeType(), osPathname)
			if de.ModeType() != os.DirMode {
				count++
			}
			return nil
		},
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			errors++
			// For the purposes of this example, a simple SkipNode will suffice,
			// although in reality perhaps additional logic might be called for.
			return godirwalk.SkipNode
		},
		Unsorted: true, // set true for faster yet non-deterministic enumeration (see godoc)
	})
	return count, errors, err
}

func iterateStoreKeys() error {
	i := 0
	err := store.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if strings.HasSuffix(string(k), "/dockerfile-content") {
				fmt.Printf("key=%s\n", k)
				i++
			}
		}
		return nil
	})
	fmt.Println("count:", i)
	return err
}

func iterateStoreKV() {
	err := store.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				if strings.HasSuffix(string(k), "//dockerfile-content") {
					// vStr, err := decompress(v)
					// if err != nil {
					// 	return err
					// }
					outputDir := fmt.Sprintf("%s", strings.Replace(string(k), "//dockerfile-content", "", -1))
					outputDir = filepath.Join("..", "datasets", "hub.docker.com", outputDir)
					// fmt.Printf("key=%s, outputDir=%s, value=%s\n", k, outputDir, v)
					fmt.Printf("key=%s, outputDir=%s\n", k, outputDir)
					err := ensureDir(outputDir)
					if err != nil {
						return err
					}
					err = ioutil.WriteFile(outputDir+"/Dockerfile", v, 0755)
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
