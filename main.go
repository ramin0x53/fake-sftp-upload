package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var host Host
var loc *time.Location
var copiedMegaBytes float64

type Host struct {
	User   string
	Pass   string
	Remote string
	Port   string
}

func (c *Host) Copy(src, desPath string) error {

	config := &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.Pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", c.Remote+":"+c.Port, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return err
	}
	defer client.Close()

	file := filepath.Base(src)

	dstFile, err := os.Create(desPath + "/" + file)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	srcFile, err := client.Open(src)
	if err != nil {
		return err
	}

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	megabytes := float64(bytes) / 1024 / 1024
	copiedMegaBytes += megabytes

	err = dstFile.Sync()
	if err != nil {
		return err
	}
	return nil
}

func CopyAndRemove(src, desPath string) error {

	err := host.Copy(src, desPath)
	if err != nil {
		return err
	}
	file := filepath.Base(src)

	err = os.Remove(desPath + "/" + file)
	if err != nil {
		return err
	}
	return nil
}

func FakeUpload(src, desPath string, count int) error {
	fmt.Println("(" + time.Now().In(loc).Format("2006-1-2 15:4:5") + ") " + "fake upload starting...\n")
	copiedMegaBytes = 0
	for i := 0; i < count; i++ {
		fmt.Print("\033[1A\033[K")
		percent := float32(i) / float32(count) * 100
		fmt.Printf("%f%%: %f Megabyte copied\n", percent, copiedMegaBytes)
		err := CopyAndRemove(src, desPath)
		if err != nil {
			return err
		}
	}
	fmt.Print("\033[1A\033[K")
	fmt.Printf("%f%%: %f Megabyte copied\n", 100.0, copiedMegaBytes)
	fmt.Println("(" + time.Now().In(loc).Format("2006-1-2 15:4:5") + ") " + "fake upload finished")
	return nil
}

func main() {
	user := flag.String("u", "", "username")
	pass := flag.String("pa", "", "password")
	remote := flag.String("r", "", "remote host")
	port := flag.String("p", "22", "remote port")
	src := flag.String("s", "", "source file")
	desPath := flag.String("dp", "/tmp", "destination path")
	count := flag.Int("c", 10, "count")
	times := flag.String("t", "04:00", "times")

	flag.Parse()

	host = Host{
		User:   *user,
		Pass:   *pass,
		Remote: *remote,
		Port:   *port,
	}

	timeZone := "Asia/Tehran"
	var err error
	loc, err = time.LoadLocation(timeZone)
	if err != nil {
		log.Fatal(err)
	}

	s := gocron.NewScheduler(loc)
	job, err := s.Every(1).Day().At(*times).Do(FakeUpload, *src, *desPath, *count)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Scheduled at: " + job.ScheduledAtTime())
	s.StartBlocking()
}
