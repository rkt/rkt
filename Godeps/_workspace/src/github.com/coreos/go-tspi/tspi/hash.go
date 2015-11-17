package tspi

// #include <trousers/tss.h>
import "C"

// Hash is a TSS hash
type Hash struct {
	handle  C.TSS_HHASH
	context C.TSS_HCONTEXT
}

// Update updates a TSS hash with the data provided. It returns an error on
// failure.
func (hash *Hash) Update(data []byte) error {
	err := tspiError(C.Tspi_Hash_UpdateHashValue(hash.handle, (C.UINT32)(len(data)), (*C.BYTE)(&data[0])))
	return err
}

// Verify checks whether a hash matches the signature signed with the
// provided key. It returns an error on failure.
func (hash *Hash) Verify(key *Key, signature []byte) error {
	err := tspiError(C.Tspi_Hash_VerifySignature(hash.handle, key.handle, (C.UINT32)(len(signature)), (*C.BYTE)(&signature[0])))
	return err
}
