// Copyright 2020 The LevelDB-Go and Pebble Authors. All rights reserved. Use
// of this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

// +build make_test_find_db

// Run using: go run -tags make_test_find_db make_test_find_db.go
package main

import (
	"log"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/sstable"
	"github.com/cockroachdb/pebble/vfs"
)

type db struct {
	db *pebble.DB
}

func open(fs vfs.FS, dir string) *db {
	d, err := pebble.Open(dir, &pebble.Options{
		Cleaner:       pebble.ArchiveCleaner{},
		EventListener: pebble.MakeLoggingEventListener(pebble.DefaultLogger),
		FS:            fs,
	})
	if err != nil {
		log.Fatal(err)
	}
	return &db{
		db: d,
	}
}

func (d *db) close() {
	if err := d.db.Close(); err != nil {
		log.Fatal(err)
	}
}

func (d *db) set(key, value string) {
	if err := d.db.Set([]byte(key), []byte(value), nil); err != nil {
		log.Fatal(err)
	}
}

func (d *db) merge(key, value string) {
	if err := d.db.Merge([]byte(key), []byte(value), nil); err != nil {
		log.Fatal(err)
	}
}

func (d *db) delete(key string) {
	if err := d.db.Delete([]byte(key), nil); err != nil {
		log.Fatal(err)
	}
}

func (d *db) singleDelete(key string) {
	if err := d.db.SingleDelete([]byte(key), nil); err != nil {
		log.Fatal(err)
	}
}

func (d *db) deleteRange(start, end string) {
	if err := d.db.DeleteRange([]byte(start), []byte(end), nil); err != nil {
		log.Fatal(err)
	}
}

func (d *db) ingest(keyVals ...string) {
	const path = "testdata/ingest.tmp"

	if len(keyVals)%2 != 0 {
		log.Fatalf("even number of key/values required")
	}

	fs := vfs.Default
	f, err := fs.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	w := sstable.NewWriter(f, sstable.WriterOptions{})

	for i := 0; i < len(keyVals); i += 2 {
		key := keyVals[i]
		value := keyVals[i+1]
		if err := w.Set([]byte(key), []byte(value)); err != nil {
			log.Fatal(err)
		}
	}

	if err := w.Close(); err != nil {
		log.Fatal(err)
	}

	if err := d.db.Ingest([]string{path}); err != nil {
		log.Fatal(err)
	}

	if err := fs.Remove(path); err != nil {
		// TODO(peter): why is the remove sometimes failing under CI?
		log.Print(err)
	}
}

func (d *db) flush() {
	if err := d.db.Flush(); err != nil {
		log.Fatal(err)
	}
}

func (d *db) compact(start, end string) {
	if err := d.db.Compact([]byte(start), []byte(end)); err != nil {
		log.Fatal(err)
	}
}

func (d *db) snapshot() *pebble.Snapshot {
	return d.db.NewSnapshot()
}

func main() {
	const dir = "testdata/find-db"

	fs := vfs.Default
	if err := fs.RemoveAll(dir); err != nil {
		log.Fatal(err)
	}

	d := open(fs, dir)
	defer d.close()

	d.set("aaa", "1")
	d.set("bbb", "2")
	d.merge("ccc", "3")
	d.merge("ccc", "4")
	d.merge("ccc", "5")
	d.flush()
	d.compact("a", "z")

	defer d.snapshot().Close()

	d.ingest("bbb", "22", "ccc", "6")
	d.ingest("ddd", "33")
	d.compact("a", "z")

	defer d.snapshot().Close()

	d.delete("aaa")
	d.singleDelete("ccc")
	d.deleteRange("bbb", "eee")
	d.flush()

	d.compact("a", "z")
}
