package main

import (
	"fmt"
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
			if de.ModeType() != os.ModeDir {
				if debug {
					log.Printf("%s %s\n", de.ModeType(), osPathname)
				}
				if osPathname != ".git" {
					count++
				}
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

func iterateStoreKV2() error {
	i := 0
	err := store.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte("github.com")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				// log.Printf("key=%s\n", k)
				if strings.HasSuffix(string(k), "//docker-content") {
					vx, err := decompress(v)
					if err != nil {
						return err
					}

					dir, filename := filepath.Split(strings.Replace(string(k), "//docker-content", "", -1))
					log.Println("Dir:", dir)       //Dir: /some/path/to/remove/
					log.Println("File:", filename) //File: ile.name

					outputDir := filepath.Join("..", "datasets", dir)

					// fmt.Printf("key=%s, outputDir=%s, value=%s\n", k, outputDir, v)
					log.Printf("key=%s, outputDir=%s filename=%s\n", k, outputDir, filename)
					err = ensureDir(outputDir)
					if err != nil {
						return err
					}

					f, err := os.Create(outputDir + "/" + filename)
					if err != nil {
						return err
					}
					bytes, err := f.Write(vx)
					if err != nil {
						return err
					}
					fmt.Printf("wrote %d bytes\n", bytes)
					f.Sync()
					f.Close()

				}
				i++
				return nil
			})
			if err != nil {
				return err
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
					outputDir = filepath.Join("..", "datasets", outputDir)
					// fmt.Printf("key=%s, outputDir=%s, value=%s\n", k, outputDir, v)
					fmt.Printf("key=%s, outputDir=%s\n", k, outputDir)
					err := ensureDir(outputDir)
					if err != nil {
						return err
					}
					// err = ioutil.WriteFile(outputDir+"/Dockerfile", v, 0755)
					f, err := os.Create(outputDir + "/Dockerfile")
					if err != nil {
						return err
					}
					bytes, err := f.Write(v)
					if err != nil {
						return err
					}
					fmt.Printf("wrote %d bytes\n", bytes)
					f.Sync()
					f.Close()

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
