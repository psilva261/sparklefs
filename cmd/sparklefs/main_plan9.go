package main

import (
	"fmt"
	"github.com/knusbaum/go9p"
	"github.com/psilva261/sparklefs/logger"
	"io"
	"os"
	"syscall"
)

func Init() (err error) {
	mtpt = "/mnt/mycel"
	if htm != "" || len(js) > 0 {
		log.Printf("not loading url/htm/js from mtpt")
		return
	}
	bs, err := os.ReadFile(mtpt + "/url")
	if err != nil {
		return
	}
	url = string(bs)
	bs, err = os.ReadFile(mtpt + "/html")
	if err != nil {
		return
	}
	htm = string(bs)
	ds, err := os.ReadDir(mtpt + "/js")
	if err != nil {
		return
	}
	for i := 0; i < len(ds); i++ {
		fn := fmt.Sprintf(mtpt+"/js/%v.js", i)
		log.Printf("fn=%v", fn)
		bs, err := os.ReadFile(fn)
		if err != nil {
			return fmt.Errorf("read all: %w", err)
		}
		js = append(js, string(bs))
	}
	return
}

func open(fn string) (rwc io.ReadWriteCloser, err error) {
	return os.OpenFile(mtpt+"/"+fn, os.O_RDWR, 0600)
}

func stat() (ok bool) {
	_, err := os.Stat(mtpt)
	return err == nil
}

func post(srv go9p.Srv) (err error) {
	f1, f2, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("pipe: %w", err)
	}

	go func() {
		err = go9p.ServeReadWriter(f1, f1, srv)
		if err != nil {
			log.Printf("serve rw: %v", err)
		}
	}()

	if err = syscall.Mount(int(f2.Fd()), -1, "/mnt/sparkle", syscall.MCREATE, ""); err != nil {
		return fmt.Errorf("mount: %w", err)
	}
	return
}

func callSparkleCtl() (rwc io.ReadWriteCloser, err error) {
	return os.OpenFile("/mnt/sparkle/ctl", os.O_RDWR, 0600)
}
