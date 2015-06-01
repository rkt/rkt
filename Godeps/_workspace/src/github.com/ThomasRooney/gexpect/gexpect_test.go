package gexpect

import (
	"strings"
	"testing"
)

func TestHelloWorld(t *testing.T) {
	t.Logf("Testing Hello World... ")
	child, err := Spawn("echo \"Hello World\"")
	if err != nil {
		t.Fatal(err)
	}
	err = child.Expect("Hello World")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDoubleHelloWorld(t *testing.T) {
	t.Logf("Testing Double Hello World... ")
	child, err := Spawn(`sh -c "echo Hello World ; echo Hello ; echo Hi"`)
	if err != nil {
		t.Fatal(err)
	}
	err = child.Expect("Hello World")
	if err != nil {
		t.Fatal(err)
	}
	err = child.Expect("Hello")
	if err != nil {
		t.Fatal(err)
	}
	err = child.Expect("Hi")
	if err != nil {
		t.Fatal(err)
	}
}

func TestHelloWorldFailureCase(t *testing.T) {
	t.Logf("Testing Hello World Failure case... ")
	child, err := Spawn("echo \"Hello World\"")
	if err != nil {
		t.Fatal(err)
	}
	err = child.Expect("YOU WILL NEVER FIND ME")
	if err != nil {
		return
	}
	t.Fatal("Expected an error for TestHelloWorldFailureCase")
}

func TestBiChannel(t *testing.T) {
	t.Logf("Testing BiChannel screen... ")
	child, err := Spawn("cat")
	if err != nil {
		t.Fatal(err)
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
	sender <- "echo\n"
	wait("echo")
	sender <- "echo2"
	wait("echo2")
	child.Close()
	// child.Wait()
}

func TestCommandStart(t *testing.T) {
	t.Logf("Testing Command... ")

	// Doing this allows you to modify the cmd struct prior to execution, for example to add environment variables
	child, err := Command("echo 'Hello World'")
	if err != nil {
		t.Fatal(err)
	}
	child.Start()
	child.Expect("Hello World")
}

var regexMatchTests = []struct {
	re   string
	good string
	bad  string
}{
	{`a`, `a`, `b`},
	{`.b`, `ab`, `ac`},
	{`a+hello`, `aaaahello`, `bhello`},
	{`(hello|world)`, `hello`, `unknown`},
	{`(hello|world)`, `world`, `unknown`},
}

func TestRegexMatch(t *testing.T) {
	t.Logf("Testing Regular Expression Matching... ")
	for _, tt := range regexMatchTests {
		runTest := func(input string) bool {
			var match bool
			child, err := Spawn("echo \"" + input + "\"")
			if err != nil {
				t.Fatal(err)
			}
			match, err = child.ExpectRegex(tt.re)
			if err != nil {
				t.Fatal(err)
			}
			return match
		}
		if !runTest(tt.good) {
			t.Errorf("Regex Not matching [%#q] with pattern [%#q]", tt.good, tt.re)
		}
		if runTest(tt.bad) {
			t.Errorf("Regex Matching [%#q] with pattern [%#q]", tt.bad, tt.re)
		}
	}
}

var regexFindTests = []struct {
	re      string
	input   string
	matches []string
}{
	{`he(l)lo wo(r)ld`, `hello world`, []string{"hello world", "l", "r"}},
	{`(a)`, `a`, []string{"a", "a"}},
	{`so.. (hello|world)`, `so.. hello`, []string{"so.. hello", "hello"}},
	{`(a+)hello`, `aaaahello`, []string{"aaaahello", "aaaa"}},
	{`\d+ (\d+) (\d+)`, `123 456 789`, []string{"123 456 789", "456", "789"}},
}

func TestRegexFind(t *testing.T) {
	t.Logf("Testing Regular Expression Search... ")
	for _, tt := range regexFindTests {
		runTest := func(input string) []string {
			child, err := Spawn("echo \"" + input + "\"")
			if err != nil {
				t.Fatal(err)
			}
			matches, err := child.ExpectRegexFind(tt.re)
			if err != nil {
				t.Fatal(err)
			}
			return matches
		}
		matches := runTest(tt.input)
		if len(matches) != len(tt.matches) {
			t.Fatalf("Regex not producing the expected number of patterns.. got[%d] ([%s]) expected[%d] ([%s])",
				len(matches), strings.Join(matches, ","),
				len(tt.matches), strings.Join(tt.matches, ","))
		}
		for i, _ := range matches {
			if matches[i] != tt.matches[i] {
				t.Errorf("Regex Expected group [%s] and got group [%s] with pattern [%#q] and input [%s]",
					tt.matches[i], matches[i], tt.re, tt.input)
			}
		}
	}
}
