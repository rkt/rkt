package metadata

import "fmt"

const (
	SvcIP      = "169.254.169.255"
	SvcPubPort = 80
	SvcPrvPort = 2375
)

func SvcPrvURL() string {
	return fmt.Sprintf("http://127.0.0.1:%v", SvcPrvPort)
}

func SvcPubURL() string {
	return fmt.Sprintf("http://%v:%v", SvcIP, SvcPubPort)
}
