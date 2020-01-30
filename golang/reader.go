package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	// "sync"

	badger "github.com/dgraph-io/badger"
	"github.com/karrick/godirwalk"
	"github.com/nozzle/throttler"
	"github.com/pkg/errors"
)

func filenameWithoutExtension(fn string) string {
	return strings.TrimSuffix(fn, path.Ext(fn))
}

func countDockerfiles(dirname string) (int, int, error) {
	count := 0
	errors := 0
	err := godirwalk.Walk(dirname, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if de.ModeType() != os.ModeDir {
				if debug {
					log.Printf("%s %s\n", de.ModeType(), osPathname)
				}
				if osPathname != ".git" || strings.HasSuffix(osPathname, ".json") || strings.HasSuffix(osPathname, ".yaml") {
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
				fmt.Printf("i=%d key=%s\n", i, k)
				i++
			}
		}
		return nil
	})
	fmt.Println("count:", i)
	return err
}

func writeDockerFile(k, v []byte) {

}

func writeDockerMeta(k, v []byte) {

}

func iterateStoreKV2(prefix, suffix string) error {
	i := 0
	err := store.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.IteratorOptions{
			PrefetchValues: true,
			PrefetchSize:   1000,
			Reverse:        false,
			AllVersions:    false,
		})
		defer it.Close()
		prefix := []byte(prefix)

		// waitGroup := &sync.WaitGroup{}
		// waitGroup.Add(1000)
		t := throttler.New(200, 1000000)

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				log.Printf("i=%d key=%s hasSuffix=%t\n", i, k, strings.HasSuffix(string(k), suffix))
				if strings.HasSuffix(string(k), suffix) {
					vx, err := decompress(v)
					if err != nil {
						return err
					}

					go func(k, vx []byte) error {
						// defer waitGroup.Done()
						defer t.Done(nil)
						// write dockerfile
						var dir, filename string
						if string(prefix) == "hub.docker.com" {
							dir = strings.Replace(string(k), suffix, "", -1)
							filename = "Dockerfile"
						} else {
							// github.com
							dir, filename = filepath.Split(strings.Replace(string(k), suffix, "", -1))
						}
						if debug {
							log.Println("Dir:", dir)       //Dir: /some/path/to/remove/
							log.Println("File:", filename) //File: ile.name
						}
						outputDir := filepath.Join("..", "..", "dockerfiles-search", dir)

						if debug {
							log.Printf("key=%s, outputDir=%s filename=%s\n", k, outputDir, filename)
						}
						err = ensureDir(outputDir)
						if err != nil {
							return errors.Wrap(err, "ensureDir")
						}
						osPathname := outputDir + "/" + filename
						f, err := os.Create(osPathname)
						if err != nil {
							return errors.Wrap(err, "Create.osPathname")
						}
						bytes, err := f.Write(vx)
						if err != nil {
							return errors.Wrap(err, "Write.osPathname")
						}
						if debug {
							log.Printf("wrote %d bytes\n", bytes)
						}
						f.Sync()
						f.Close()

						isDockerfileParser := true
						if isDockerfileParser {
							jsonBytes, err := dockerfileParser(osPathname)
							if err != nil {
								return nil
							}

							// write dockerfile metafile
							filename = filenameWithoutExtension(filename) + ".meta.yaml"
							f2, err := os.Create(outputDir + "/" + filename)
							if err != nil {
								return errors.Wrap(err, "Create.meta")
							}
							bytes2, err := f2.Write(jsonBytes)
							if err != nil {
								return errors.Wrap(err, "Write.meta")
							}
							if debug {
								log.Printf("wrote %d bytes2\n", bytes2)
							}
							f2.Sync()
							f2.Close()
						}
						return nil
					}(k, vx)
					// index into elasticsearch

					i++
				}
				t.Throttle()
				// waitGroup.Wait()
				return nil
			})
			if err != nil {
				return err
			}
			if t.Err() != nil {
				// Loop through the errors to see the details
				for i, err := range t.Errs() {
					log.Printf("error #%d: %s", i, err)
				}
				log.Fatal(t.Err())
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
					vx, err := decompress(v)
					if err != nil {
						return err
					}
					outputDir := fmt.Sprintf("%s", strings.Replace(string(k), "//dockerfile-content", "", -1))
					outputDir = filepath.Join("..", "datasets", outputDir)
					// fmt.Printf("key=%s, outputDir=%s, value=%s\n", k, outputDir, v)
					fmt.Printf("key=%s, outputDir=%s\n", k, outputDir)
					err = ensureDir(outputDir)
					if err != nil {
						return err
					}
					// err = ioutil.WriteFile(outputDir+"/Dockerfile", v, 0755)
					f, err := os.Create(outputDir + "/Dockerfile")
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
