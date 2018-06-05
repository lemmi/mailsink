// Package main provides a simple mail logger
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/bradfitz/go-smtpd/smtpd"
)

type buf struct {
	bytes.Buffer
}

func (b buf) Close() error {
	return nil
}

type env struct {
	dir  string
	host string

	from  smtpd.MailAddress
	rcpts []smtpd.MailAddress
	to    string
	body  bytes.Buffer
}

func (e *env) AddRecipient(rcpt smtpd.MailAddress) error {
	e.rcpts = append(e.rcpts, rcpt)
	return nil
}

func (e *env) BeginData() error {
	log.Printf("Processing new mail from %q", e.from.Email())

	if len(e.rcpts) == 0 {
		return smtpd.SMTPError("554 5.5.1 Error: no valid recipients")
	}

	for _, r := range e.rcpts {
		log.Printf("%q == %q", r.Hostname(), e.host)
		if r.Hostname() == e.host {
			log.Printf("Found %s", r.Email())
			e.to = r.Email()
			return nil
		}
	}

	return smtpd.SMTPError("554 5.5.1 Error: no valid recipients")
}

func (e *env) Write(line []byte) error {
	_, err := e.body.Write(line)
	return err
}

func (e *env) Close() error {
	if err := os.Mkdir(e.path(), 0755); err != nil {
		if !os.IsExist(err) {
			log.Panicln(err)
			return err
		}
	}
	if err := ioutil.WriteFile(e.filename(), e.body.Bytes(), 0644); err != nil {
		log.Panicln(err)
		return err
	}
	return nil
}

func (e *env) path() string {
	return filepath.Join(e.dir, e.to)
}
func (e *env) filename() string {
	return fmt.Sprintf("%s_%s.eml", time.Now().UTC().Format(time.RFC3339), e.from.Email())
}
func (e *env) filepath() string {
	return filepath.Join(e.path(), e.filename())
}

type Config struct {
	Address string
	Dir     string
	Host    string
	//Whitelist []string
}

func main() {
	opts := struct {
		addr string
		dir  string
		host string
	}{}

	flag.StringVar(&opts.addr, "addr", ":2525", "Address to listen on")
	flag.StringVar(&opts.dir, "dir", "", "Directory to log messages to")
	flag.StringVar(&opts.host, "host", "", "Hostname to accept")
	flag.Parse()

	if opts.host == "" {
		log.Panicln("No host set")
	}

	s := smtpd.Server{
		Addr: opts.addr,
		OnNewMail: func(c smtpd.Connection, from smtpd.MailAddress) (smtpd.Envelope, error) {
			return &env{
				from: from,
				dir:  opts.dir,
				host: opts.host,
			}, nil
		},
	}
	err := s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
