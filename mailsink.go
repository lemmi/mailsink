// Package main provides a simple mail logger
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
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
	from  smtpd.MailAddress
	rcpts []smtpd.MailAddress
	body  io.WriteCloser
}

func (e *env) AddRecipient(rcpt smtpd.MailAddress) error {
	e.rcpts = append(e.rcpts, rcpt)
	return nil
}

func (e *env) BeginData() error {
	if len(e.rcpts) == 0 {
		return smtpd.SMTPError("554 5.5.1 Error: no valid recipients")
	}

	return nil
}

func (e *env) Write(line []byte) error {
	_, err := e.body.Write(line)
	return err
}

func (e *env) Close() error {
	if b, ok := e.body.(fmt.Stringer); ok {
		fmt.Println(b.String())
	}
	return e.body.Close()
}

type fileLogger struct {
	path string
}

func (f fileLogger) New(from smtpd.MailAddress) (io.WriteCloser, error) {
	path := filepath.Join(f.path, time.Now().UTC().Format(time.RFC3339), "_", from.Email())
	return os.Create(path)
}

func stdoutLogger(smtpd.MailAddress) (io.WriteCloser, error) {
	return new(buf), nil
}

type logFunc func(from smtpd.MailAddress) (io.WriteCloser, error)

func main() {
	opts := struct {
		addr string
		dir  string
	}{}

	flag.StringVar(&opts.addr, "addr", ":2525", "Address to listen on")
	flag.StringVar(&opts.dir, "dir", "", "Directory to log messages to")
	flag.Parse()

	var newBodyFunc logFunc = stdoutLogger
	if opts.dir != "" {
		fl := fileLogger{opts.dir}
		newBodyFunc = fl.New
	}

	s := smtpd.Server{
		Addr: opts.addr,
		OnNewMail: func(c smtpd.Connection, from smtpd.MailAddress) (smtpd.Envelope, error) {
			body, err := newBodyFunc(from)
			return &env{
				from: from,
				body: body,
			}, err
		},
	}
	err := s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
