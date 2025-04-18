package database

import (
	"db/btree"
	"db/cache"
	"fmt"
	"path/filepath"
)

// For Dev Nigger : Collection is basically a wrapper around a single B-tree instance
// plus metadata like its name and base directory
type Collection struct {
	name    string
	order   int
	btree   *btree.BTree
	baseDir string
}

// InsertKV wraps the btree insert
func (c *Collection) InsertKV(key string, value interface{}) {
	err := c.btree.Insert(key, value)
	if err != nil {
		panic(fmt.Sprintf("Failed to insert key %s into collection %s: %v", key, c.name, err))
	}
	fmt.Printf("Inserted key: %s (value: %v) into collection: %s\n", key, value, c.name)
	cache.InsertInCacheMemory(filepath.Dir(c.baseDir), c.name, key, value.(string))
}

// FindKey wraps the btree find
func (c *Collection) FindKey(key string) (interface{}, bool) {
	value, err := cache.FindInCacheMemory(filepath.Dir(c.baseDir), c.name, key)
	var val interface{} = value
	found := false
	if err == nil {
		found = true
	} else {
		val, found, err = c.btree.Find(key)
		if err != nil {
			panic(fmt.Sprintf("Failed to find key %s in collection %s: %v", key, c.name, err))
		}
	}
	if found {
		fmt.Printf("Found key: %s => %v (in collection: %s)\n", key, val, c.name)
	} else {
		fmt.Printf("Key not found: %s (in collection: %s)\n", key, c.name)
	}
	return val, found
}

func (c *Collection) FindAllKV() []btree.KeyValue {
	result := c.btree.FindAll()
	return result
}

// UpdateKV wraps the btree update
func (c *Collection) UpdateKV(key string, value interface{}) {
	updated, err := c.btree.Update(key, value)
	if err != nil {
		panic(fmt.Sprintf("Failed to update key %s in collection %s: %v", key, c.name, err))
	}
	if updated {
		fmt.Printf("Updated key: %s => %v (in collection: %s)\n", key, value, c.name)
	} else {
		fmt.Printf("Key not found for update: %s (in collection: %s), inserting...\n", key, c.name)
		// Insert
		if err := c.btree.Insert(key, value); err != nil {
			panic(fmt.Sprintf("Failed to insert key %s after update attempt: %v", key, err))
		}
	}
	cache.UpdateCacheInMemory(filepath.Dir(c.baseDir), c.name, key, value.(string))
}

// DeleteKey wraps the btree delete
func (c *Collection) DeleteKey(key string) {
	deleted, err := c.btree.Delete(key)
	if err != nil {
		panic(fmt.Sprintf("Failed to delete key %s in collection %s: %v", key, c.name, err))
	}
	if deleted {
		fmt.Printf("Deleted key: %s (in collection: %s)\n", key, c.name)
	} else {
		fmt.Printf("Key not found for deletion: %s (in collection: %s)\n", key, c.name)
	}
	cache.DeleteFromCacheMemory(filepath.Dir(c.baseDir), c.name, key)
}
