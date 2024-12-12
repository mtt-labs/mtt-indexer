package db

import "encoding/binary"

func (l *LDB) getU64(key string) (uint64, error) {
	data, err := l.DB.Get([]byte(key), nil)
	if err != nil {
		return 0, err
	}
	return BytesToUint64(data), nil
}

func Uint64ToBytes(i uint64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	return buf
}

func BytesToUint64(buf []byte) uint64 {
	return binary.BigEndian.Uint64(buf)
}
