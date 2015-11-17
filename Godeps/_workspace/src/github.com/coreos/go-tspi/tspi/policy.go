package tspi

// #include <trousers/tss.h>
import "C"

// Policy is a TSS policy object
type Policy struct {
	handle  C.TSS_HPOLICY
	context C.TSS_HCONTEXT
}

// SetSecret sets the secret for a policy. This policy may then be applied to
// another object.
func (policy *Policy) SetSecret(sectype int, secret []byte) error {
	err := tspiError(C.Tspi_Policy_SetSecret(policy.handle, (C.TSS_FLAG)(sectype), (C.UINT32)(len(secret)), (*C.BYTE)(&secret[0])))
	return err
}
