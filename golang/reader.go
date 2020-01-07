package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	badger "github.com/dgraph-io/badger"
)

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
					fmt.Printf("key=%s, outputDir=%s, value=%s\n", k, outputDir, v)
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
