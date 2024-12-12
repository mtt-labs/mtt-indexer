package db

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/shopspring/decimal"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	_ "github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"mtt-indexer/logger"
	"mtt-indexer/types"
	"os"
	"reflect"
	"sync"
)

const dbName = "mtt_index_"
const indexKey = "index"

type LDB struct {
	DB   *leveldb.DB
	lock sync.RWMutex
}

func NewLdb(tailFix string) *LDB {
	l := &LDB{}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	db, err := leveldb.OpenFile(homeDir+"/."+dbName+tailFix, nil)
	if err != nil {
		panic(err)
	}
	l.DB = db
	l.lock = sync.RWMutex{}
	return l
}

func (l *LDB) Transaction(fc func(l *LDB, batch *leveldb.Batch) error) error {
	batch := new(leveldb.Batch)
	l.lock.Lock()
	defer l.lock.Unlock()
	err := fc(l, batch)
	if err != nil {
		return err
	}

	return l.DB.Write(batch, nil)
}

func (l *LDB) GetRecordByType(record types.DbRecord) (interface{}, error) {
	key := []byte(record.Key())
	data, err := l.DB.Get(key, nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return 1, nil
		}
		return nil, err
	}

	//recordType := reflect.TypeOf(record).Elem()
	//newRecordPtr := reflect.New(recordType)
	//newRecord := newRecordPtr.Interface()
	recordPtr := reflect.New(reflect.TypeOf(record).Elem()).Interface()
	err = json.Unmarshal(data, recordPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %v", err)
	}

	return recordPtr, nil
}

func getRecordType(record interface{}) string {
	t := reflect.TypeOf(record)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

func (l *LDB) GetAllRecordsWithAutoId(record types.DbRecordAutoId, limit, offset int, ascending bool) ([]interface{}, int, error) {
	if limit <= 0 {
		return nil, 0, fmt.Errorf("limit must be greater than 0")
	}
	if offset < 0 {
		return nil, 0, fmt.Errorf("offset cannot be negative")
	}

	var records []interface{}
	prefix := []byte(record.Prefix())

	l.lock.RLock()
	defer l.lock.RUnlock()

	// Create an iterator for the prefix
	iter := l.DB.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()

	// Count total records
	total := 0
	for iter.Seek(prefix); iter.Valid(); iter.Next() {
		total++
	}
	if err := iter.Error(); err != nil {
		logger.Logger.Errorf("iterator error during total count: %v", err)
		return nil, 0, err
	}

	iter.Release()
	iter = l.DB.NewIterator(util.BytesPrefix(prefix), nil)
	if !ascending {
		iter.Last() // Start from the end if descending
	}
	defer iter.Release()

	count := 0
	started := 0

	for {
		if len(iter.Value()) == 0 {
			break
		}

		if started < offset { // Skip until the offset
			started++
			continue
		}

		if count >= limit { // Stop if we reached the limit
			break
		}

		data := iter.Value()

		recordType := reflect.TypeOf(record).Elem()
		newRecordPtr := reflect.New(recordType)
		newRecord := newRecordPtr.Interface()

		err := json.Unmarshal(data, &newRecord)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal record: %v", err)
		}

		records = append(records, newRecord)
		count++

		if ascending {
			if !iter.Next() { // Move forward
				break
			}
		} else {
			if !iter.Prev() { // Move backward
				break
			}
		}
	}

	if err := iter.Error(); err != nil {
		logger.Logger.Errorf("iterator error: %v", err)
		return nil, 0, err
	}
	return records, total, nil
}

func getNextID(db *leveldb.DB, recordType string) (uint64, error) {
	key := autoIncrementKey(recordType)

	data, err := db.Get([]byte(key), nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return 1, nil
		}
		return 0, err
	}

	return binary.BigEndian.Uint64(data) + 1, nil
}

func autoIncrementKey(recordType string) string {
	return fmt.Sprintf("auto_increment_%s", recordType)
}

func storeRecordWithAutoID(db *leveldb.DB, batch *leveldb.Batch, record types.DbRecordAutoId) error {
	nextID, err := getNextID(db, record.Prefix())
	if err != nil {
		return err
	}

	record.SetId(nextID)

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	key := record.Key()

	batch.Put([]byte(key), data)

	idData := make([]byte, 8)
	binary.BigEndian.PutUint64(idData, nextID)
	batch.Put([]byte(autoIncrementKey(record.Prefix())), idData)
	return nil
}

func StoreRecord(db *leveldb.DB, batch *leveldb.Batch, record types.DbRecord) error {
	if recordAuto, ok := record.(types.DbRecordAutoId); ok {
		err := storeRecordWithAutoID(db, batch, recordAuto)
		if err != nil {
			return err
		}
		return nil
	} else {
		data, err := json.Marshal(record)
		if err != nil {
			return err
		}

		key := record.Key()

		batch.Put([]byte(key), data)
		return db.Write(batch, nil)
	}
}
