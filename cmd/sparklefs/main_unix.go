//go:build !plan9

package main

import (
	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"fmt"
	"github.com/knusbaum/go9p"
	"github.com/psilva261/sparklefs/logger"
	"io"
	"os/user"
)

var fsys *client.Fsys

func Init() (err error) {
	log.Printf("Init...")
	conn, err := client.DialService("opossum")
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	u, err := user.Current()
	if err != nil {
		return
	}
	un := u.Username
	fsys, err = conn.Attach(nil, un, "")
	if err != nil {
		log.Fatalf("attach: %v", err)
	}
	if htm != "" || len(js) > 0 {
		log.Printf("not loading htm/js from service")
		return
	}
	log.Printf("open url...")
	fid, err := fsys.Open("url", plan9.OREAD)
	if err != nil {
		return
	}
	defer fid.Close()
	bs, err := io.ReadAll(fid)
	if err != nil {
		return
	}
	url = string(bs)
	log.Printf("open html...")
	fid, err = fsys.Open("html", plan9.OREAD)
	if err != nil {
		return
	}
	defer fid.Close()
	bs, err = io.ReadAll(fid)
	if err != nil {
		return
	}
	htm = string(bs)
	log.Printf("open js...")
	dfid, err := fsys.Open("js", plan9.OREAD)
	if err != nil {
		return
	}
	defer dfid.Close()
	ds, err := dfid.Dirreadall()
	if err != nil {
		return
	}
	log.Printf("ds=%+v", ds)
	for i := 0; i < len(ds); i++ {
		fn := fmt.Sprintf("js/%v.js", i)
		log.Printf("fn=%v", fn)
		fid, err := fsys.Open(fn, plan9.OREAD)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		bs, err := io.ReadAll(fid)
		if err != nil {
			fid.Close()
			return fmt.Errorf("read all: %w", err)
		}
		js = append(js, string(bs))
		fid.Close()
	}
	return
}

func open(fn string) (rwc io.ReadWriteCloser, err error) {
	return fsys.Open(fn, plan9.ORDWR)
}

func stat() (ok bool) {
	_, err := fsys.Stat("html")
	return err == nil
}

func post(srv go9p.Srv) (err error) {
	if service == "" {
		return fmt.Errorf("no service specified")
	}
	return go9p.PostSrv(service, srv)
}
