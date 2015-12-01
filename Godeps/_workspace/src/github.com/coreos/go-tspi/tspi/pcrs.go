package tspi

// #include <trousers/tss.h>
import "C"
import "unsafe"

// PCRs is a structure representing a PCR object and the PCR values
type PCRs struct {
	handle  C.TSS_HPCRS
	context C.TSS_HCONTEXT
	pcrs    [24][]byte
	pcrset  [24]bool
}

// SetPCRs takes an array of integers referring to PCRs. Any queries performed
// with this PCR object will then query these PCRs.
func (pcrs *PCRs) SetPCRs(pcrset []int) error {
	for pcr := range pcrs.pcrset {
		pcrs.pcrset[pcr] = false
	}
	for _, pcr := range pcrset {
		err := tspiError(C.Tspi_PcrComposite_SelectPcrIndex(pcrs.handle, (C.UINT32)(pcr)))
		if err != nil {
			return err
		}
		pcrs.pcrset[pcr] = true
	}
	return nil
}

// GetPCRValues obtains the PCR values for any PCRs that have been set.
func (pcrs *PCRs) GetPCRValues() ([][]byte, error) {
	var buflen C.UINT32
	var buf *C.BYTE
	for pcr := range pcrs.pcrs {
		if pcrs.pcrset[pcr] == false {
			continue
		}
		err := tspiError(C.Tspi_PcrComposite_GetPcrValue(pcrs.handle, (C.UINT32)(pcr), &buflen, &buf))
		if err != nil {
			return nil, err
		}
		pcrs.pcrs[pcr] = C.GoBytes(unsafe.Pointer(buf), (C.int)(buflen))
		C.Tspi_Context_FreeMemory(pcrs.context, buf)
	}
	return pcrs.pcrs[:], nil
}
