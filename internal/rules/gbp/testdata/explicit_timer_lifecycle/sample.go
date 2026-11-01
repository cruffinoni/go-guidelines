package sample

import "time"

func f() {
	time.AfterFunc(time.Second, func() {}) // want "obscures timer lifecycle"
}

func g() {
	t := time.NewTicker(time.Second) // want "without a visible Stop call"
	_ = t
}

func h() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
}
