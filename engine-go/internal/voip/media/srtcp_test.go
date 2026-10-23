package media

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/alltomatos/watinkdev/engine-go/internal/voip/core"
)

func srtcpKeysFor(t *testing.T, root []byte, jid string) *SrtcpKeys {
	t.Helper()
	km, err := DerivePerJidSrtpKey(root, jid)
	if err != nil {
		t.Fatal(err)
	}
	k, err := DeriveSrtcpKeys(km)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// Vetor da meowcaller (srtp/e2e_test.go, TestDeriveE2eSRTCPKeysFromRaw...): raiz 0xa0.., participante
// "111111111111111:14@lid". Prova que os rótulos 3/4/5 e o HKDF são os do WhatsApp.
func TestDeriveSrtcpKeysMatchesReferenceVector(t *testing.T) {
	root := make([]byte, 32)
	for i := range root {
		root[i] = byte(0xa0 + i)
	}
	k := srtcpKeysFor(t, root, "111111111111111:14@lid")
	for name, c := range map[string]struct {
		got  []byte
		want string
	}{
		"cipher": {k.cipherKey, "96ec6c38976e1561a01929ef61627fd9"},
		"auth":   {k.authKey, "b033e3437b17edaa45863a19a6a3969633353bc7"},
		"salt":   {k.salt, "d66012ec5832edc623a9fd742ff1"},
	} {
		if hex.EncodeToString(c.got) != c.want {
			t.Errorf("%s = %x, want %s", name, c.got, c.want)
		}
	}
}

func TestSrtcpKeysDifferFromSrtp(t *testing.T) {
	root := bytes.Repeat([]byte{7}, 32)
	km, _ := DerivePerJidSrtpKey(root, "1:0@lid")
	rtpAuth, _ := deriveSrtpKey(km.MasterKey, km.MasterSalt, core.SRTPLabelAuth, 20)
	if bytes.Equal(srtcpKeysFor(t, root, "1:0@lid").authKey, rtpAuth) {
		t.Fatal("as chaves do SRTCP precisam usar rótulos diferentes das do SRTP")
	}
}

// Vetor do SR da meowcaller (rtcp.senderReport): 28 bytes que o SRTCP cifra a partir do byte 8.
func TestSrtcpRoundTripAndLayout(t *testing.T) {
	plain, _ := hex.DecodeString("80c8000612345678ea11180000000000000006400000000500000258")
	k := srtcpKeysFor(t, bytes.Repeat([]byte{3}, 32), "222222222222222:0@lid")
	prot, err := k.Protect(0x12345678, 1, plain)
	if err != nil {
		t.Fatal(err)
	}
	if len(prot) != len(plain)+SrtcpTrailerLen {
		t.Fatalf("len = %d, want %d", len(prot), len(plain)+SrtcpTrailerLen)
	}
	if !bytes.Equal(prot[:8], plain[:8]) {
		t.Fatal("os 8 primeiros bytes ficam em claro")
	}
	if bytes.Equal(prot[8:len(plain)], plain[8:]) {
		t.Fatal("o corpo precisa estar cifrado")
	}
	if idx := prot[len(plain) : len(plain)+4]; !bytes.Equal(idx, []byte{0x80, 0, 0, 1}) {
		t.Fatalf("índice com o bit E = %x", idx)
	}
	got, index, err := k.Unprotect(0x12345678, prot)
	if err != nil || index != 1 || !bytes.Equal(got, plain) {
		t.Fatalf("unprotect = %x %d %v", got, index, err)
	}
}

func TestSrtcpRejectsForgedAndShort(t *testing.T) {
	k := srtcpKeysFor(t, bytes.Repeat([]byte{3}, 32), "222222222222222:0@lid")
	plain := make([]byte, 28)
	plain[0] = 0x80
	prot, _ := k.Protect(1, 5, plain)

	forged := append([]byte(nil), prot...)
	forged[len(forged)-1] ^= 1
	if _, _, err := k.Unprotect(1, forged); err == nil {
		t.Error("tag adulterada foi aceita")
	}
	body := append([]byte(nil), prot...)
	body[10] ^= 1
	if _, _, err := k.Unprotect(1, body); err == nil {
		t.Error("corpo adulterado foi aceito")
	}
	other := srtcpKeysFor(t, bytes.Repeat([]byte{4}, 32), "222222222222222:0@lid")
	if _, _, err := other.Unprotect(1, prot); err == nil {
		t.Error("chave errada foi aceita")
	}
	if _, _, err := k.Unprotect(1, prot[:10]); err != ErrSrtcpShort {
		t.Errorf("curto: %v", err)
	}
}
