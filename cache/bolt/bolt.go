package bolt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/shiroyk/cloudcat/lib/logger"
	"go.etcd.io/bbolt"
)

const (
	defaultBatchSize = 100000
	DefaultPath      = "cache"
	defaultInterval  = 10 * time.Minute
	defaultKeysClean = 64
	fillPercent      = 0.9
)

var (
	expireBucketName = []byte("expire")
	// ErrKeyNotFound not found the key
	ErrKeyNotFound = errors.New("key not found")
)

// DB a bbolt.DB instance
type DB struct {
	bucketName []byte
	db         *bbolt.DB
	interval   time.Duration
	closedC    chan struct{}
}

// NewDB creates a new DB instance
// if interval above 0, will not clear expired keys
func NewDB(path, name string, interval time.Duration) (*DB, error) {
	if path == "" {
		path = DefaultPath
	}
	err := os.MkdirAll(path, 0700)
	if err != nil {
		return nil, err
	}
	db, err := bbolt.Open(filepath.Join(path, name), 0600, &bbolt.Options{
		Timeout:         1 * time.Second,
		InitialMmapSize: 1024,
	})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err = tx.CreateBucketIfNotExists([]byte(name)); err != nil {
			return err
		}
		if _, err = tx.CreateBucketIfNotExists(expireBucketName); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	c := &DB{
		bucketName: []byte(name),
		interval:   interval,
		db:         db,
		closedC:    make(chan struct{}),
	}
	go c.expire()
	return c, nil
}

// Put method writes kv according to the bucket.
func (db *DB) Put(key, value []byte) (err error) {
	return db.PutWithTimeout(key, value, 0)
}

// PutWithTimeout method writes kv with timeout according to the bucket.
func (db *DB) PutWithTimeout(key, value []byte, timeout time.Duration) (err error) {
	var tx *bbolt.Tx
	if tx, err = db.db.Begin(true); err != nil {
		return
	}
	bucket := tx.Bucket(db.bucketName)
	if err = bucket.Put(key, value); err != nil {
		_ = tx.Rollback()
		return
	}
	if timeout > 0 {
		// put the timeout to expire bucket
		expireBucket := tx.Bucket(expireBucketName)
		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.BigEndian, time.Now().Add(timeout).Unix())
		if err != nil {
			return
		}
		if err = expireBucket.Put(key, buf.Bytes()); err != nil {
			_ = tx.Rollback()
			return
		}
	}
	return tx.Commit()
}

// Get reads the value from the bucket with key.
func (db *DB) Get(key []byte) (value []byte, err error) {
	var tx *bbolt.Tx
	if tx, err = db.db.Begin(false); err != nil {
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if ddl := tx.Bucket(expireBucketName).Get(key); ddl != nil {
		// scan deadline of the key
		if time.Now().Unix() > int64(binary.BigEndian.Uint64(ddl)) {
			return nil, ErrKeyNotFound
		}
	}

	value = tx.Bucket(db.bucketName).Get(key)

	return
}

// Delete a specified key from DB.
func (db *DB) Delete(key []byte) error {
	return db.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(db.bucketName).Delete(key)
	})
}

// DeleteBatch delete data in batch.
func (db *DB) DeleteBatch(keys [][]byte) error {
	batchLoopNum := len(keys) / defaultBatchSize
	if len(keys)%defaultBatchSize > 0 {
		batchLoopNum++
	}

	for batchIdx := 0; batchIdx < batchLoopNum; batchIdx++ {
		offset := batchIdx * defaultBatchSize
		tx, err := db.db.Begin(true)
		if err != nil {
			return err
		}
		bucket := tx.Bucket(db.bucketName)
		bucket.FillPercent = fillPercent
		for itemIdx := offset; itemIdx < offset+defaultBatchSize; itemIdx++ {
			if itemIdx >= len(keys) {
				break
			}
			key := keys[itemIdx]
			if err = bucket.Delete(key); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database.
func (db *DB) Close() error {
	close(db.closedC)
	if err := db.db.Sync(); err != nil {
		return err
	}
	return db.db.Close()
}

// expire timing scan the expired keys and delete them.
func (db *DB) expire() {
	if db.interval <= 0 {
		return
	}
	ticker := time.NewTicker(db.interval)
	defer ticker.Stop()

	exitSign := make(chan os.Signal, 1)
	signal.Notify(exitSign, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case <-ticker.C:
			now := time.Now().Unix()
			tx, err := db.db.Begin(false)
			if err != nil {
				logger.Errorf("error start cleaning transactions %s", err)
				continue
			}

			bucket := tx.Bucket(expireBucketName)
			if bucket.Stats().KeyN < defaultKeysClean {
				continue
			}

			var deletedKeys [][]byte
			cursor := bucket.Cursor()

			for realKey, ddl := cursor.First(); realKey != nil; realKey, ddl = cursor.Next() {
				timeout := int64(binary.BigEndian.Uint64(ddl))
				if now > timeout {
					deletedKeys = append(deletedKeys, realKey)
				}
			}

			_ = tx.Rollback()

			if err = db.DeleteBatch(deletedKeys); err != nil {
				logger.Errorf("error cleaning expired keys %s", err)
			}

		case <-exitSign:
		case <-db.closedC:
			return
		}
	}
}
