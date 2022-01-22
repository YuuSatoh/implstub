package b

import (
	"implstub/src/a"
	"implstub/src/a/c"
)

// Hoge interface
type Hoge interface {
	// yey hogehoge
	yey(msg string, id int64) (string, error)
	// hoge
	piyo(adb a.ADB, db BDB) error
}

type Foo interface {
	// bow hogehoge.
	bow(db c.CDB) (err error)
}

func (bdb BDB) yey(msg string, id int64) (string, error) {
	panic("not implemented") // TODO: Implement
}

type BDB struct {
	name string
	Adb  a.ADB
}

type BResis struct {
}
