package gexpect

import (
	"log"
	"strings"
	"testing"
)

func TestHelloWorld(*testing.T) {
	log.Printf("Testing Hello World... ")
	child, err := Spawn("echo \"Hello World\"")
	if err != nil {
		panic(err)
	}
	err = child.Expect("Hello World")
	if err != nil {
		panic(err)
	}
	log.Printf("Success\n")
}

func TestHelloWorldFailureCase(*testing.T) {
	log.Printf("Testing Hello World Failure case... ")
	child, err := Spawn("echo \"Hello World\"")
	if err != nil {
		panic(err)
	}
	err = child.Expect("YOU WILL NEVER FIND ME")
	if err != nil {
		log.Printf("Success\n")
		return
	}
	panic("Expected an error for TestHelloWorldFailureCase")
}

func TestBiChannel(*testing.T) {
	log.Printf("Testing BiChannel screen... ")
	child, err := Spawn("screen")
	if err != nil {
		panic(err)
	}
	sender, reciever := child.AsyncInteractChannels()
	wait := func(str string) {
		for {
			msg, open := <-reciever
			if !open {
				return
			}
			if strings.Contains(msg, str) {
				return
			}
		}
	}
	sender <- "\n"
	sender <- "echo Hello World\n"
	wait("Hello World")
	sender <- "times\n"
	wait("s")
	sender <- "^D\n"
	log.Printf("Success\n")

}

func TestExpectRegex(*testing.T) {
	log.Printf("Testing ExpectRegex... ")

	child, err := Spawn("/bin/sh times")
	if err != nil {
		panic(err)
	}
	child.ExpectRegex("Name")
	log.Printf("Success\n")

}

func TestCommandStart(*testing.T) {
	log.Printf("Testing Command... ")

	// Doing this allows you to modify the cmd struct prior to execution, for example to add environment variables
	child, err := Command("echo 'Hello World'")
	if err != nil {
		panic(err)
	}
	child.Start()
	child.Expect("Hello World")
	log.Printf("Success\n")
}

func TestExpectFtp(*testing.T) {
	log.Printf("Testing Ftp... ")

	child, err := Spawn("ftp ftp.openbsd.org")
	if err != nil {
		panic(err)
	}
	child.Expect("Name")
	child.SendLine("anonymous")
	child.Expect("Password")
	child.SendLine("pexpect@sourceforge.net")
	child.Expect("ftp> ")
	child.SendLine("cd /pub/OpenBSD/3.7/packages/i386")
	child.Expect("ftp> ")
	child.SendLine("bin")
	child.Expect("ftp> ")
	child.SendLine("prompt")
	child.Expect("ftp> ")
	child.SendLine("pwd")
	child.Expect("ftp> ")
	log.Printf("Success\n")
}

func TestInteractPing(*testing.T) {
	log.Printf("Testing Ping interact... \n")

	child, err := Spawn("ping -c8 8.8.8.8")
	if err != nil {
		panic(err)
	}
	child.Interact()
	log.Printf("Success\n")

}
