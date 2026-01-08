package wa

import (
	"encoding/hex"
)

func FormatAaguidBytesToString(b []byte) string {
	if len(b) != 16 {
		b = make([]byte, 16)
	}
	return hex.EncodeToString(b)[:8] + "-" + hex.EncodeToString(b)[8:12] + "-" + hex.EncodeToString(b)[12:16] + "-" + hex.EncodeToString(b)[16:20] + "-" + hex.EncodeToString(b)[20:]
}

func (wa *WA) ResolveNameFromAaGuid(aaguid []byte) string {

	if entry, ok := wa.aaguidMap[FormatAaguidBytesToString(aaguid)]; ok {
		return entry.Name
	}
	return "Unnamed passkey"
}
