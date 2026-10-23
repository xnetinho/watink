package signaling

import (
	"bytes"
	"context"
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

func vPeer() types.JID    { return types.NewJID("111111111111111", types.HiddenUserServer) }
func vCreator() types.JID { return types.NewJID("222222222222222", types.HiddenUserServer) }

func childTagsOf(n waBinary.Node, action string) []string {
	kids, _ := n.Content.([]waBinary.Node)
	if len(kids) == 0 || kids[0].Tag != action {
		return nil
	}
	inner, _ := kids[0].Content.([]waBinary.Node)
	out := make([]string, 0, len(inner))
	for _, c := range inner {
		out = append(out, c.Tag)
	}
	return out
}

func childOf(n waBinary.Node, action, tag string) (waBinary.Node, bool) {
	kids, _ := n.Content.([]waBinary.Node)
	if len(kids) == 0 {
		return waBinary.Node{}, false
	}
	inner, _ := kids[0].Content.([]waBinary.Node)
	for _, c := range inner {
		if c.Tag == tag {
			return c, true
		}
	}
	return waBinary.Node{}, false
}

func equalTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type vidSock struct{ capSock }

func (vidSock) CreateParticipantNodes(_ context.Context, devs []types.JID, _ []byte, _ waBinary.Attrs) ([]waBinary.Node, bool, error) {
	nodes := make([]waBinary.Node, 0, len(devs))
	for _, d := range devs {
		nodes = append(nodes, waBinary.Node{Tag: "to", Attrs: waBinary.Attrs{"jid": d}, Content: []waBinary.Node{
			{Tag: "enc", Attrs: waBinary.Attrs{"v": "2", "type": "msg", "count": "0"}, Content: []byte{1}},
		}})
	}
	return nodes, false, nil
}

// O servidor devolve erro 439 se a ordem dos filhos do offer estiver errada, e o iPhone testado
// ignora um <video> com enc="vp8" ou com o atributo legado "orientation".
func TestOfferWithVideoOrderAndAttributes(t *testing.T) {
	n, err := BuildOfferStanza(context.Background(), vidSock{}, "CALL1", bytes.Repeat([]byte{1}, 32), vPeer(), true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"audio", "audio", "video", "net", "capability", "destination", "encopt"}
	if got := childTagsOf(n, "offer"); !equalTags(got, want) {
		t.Fatalf("filhos do offer = %v, esperado %v", got, want)
	}
	v, _ := childOf(n, "offer", "video")
	if v.Attrs["enc"] != "h.264" || v.Attrs["dec"] != "H264" {
		t.Fatalf("<video> do offer enc=%v dec=%v, esperado h.264/H264", v.Attrs["enc"], v.Attrs["dec"])
	}
	if _, has := v.Attrs["orientation"]; has {
		t.Fatal("o atributo legado orientation não pode existir")
	}
	for _, k := range []string{"screen_width", "screen_height", "device_orientation"} {
		if _, ok := v.Attrs[k]; !ok {
			t.Fatalf("<video> sem %s", k)
		}
	}
}

func TestOfferWithoutVideoHasNoVideoChild(t *testing.T) {
	n, err := BuildOfferStanza(context.Background(), vidSock{}, "CALL1", bytes.Repeat([]byte{1}, 32), vPeer(), false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := childOf(n, "offer", "video"); ok {
		t.Fatal("oferta de voz não pode ter <video>")
	}
}

func TestAcceptWithVideo(t *testing.T) {
	n, err := BuildAcceptStanza(context.Background(), vidSock{}, "CALL1", bytes.Repeat([]byte{1}, 32), vPeer(), vCreator(), true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"audio", "video", "net", "enc", "encopt"}
	if got := childTagsOf(n, "accept"); !equalTags(got, want) {
		t.Fatalf("filhos do accept = %v, esperado %v", got, want)
	}
	v, _ := childOf(n, "accept", "video")
	if v.Attrs["dec"] != "H264" || v.Attrs["device_orientation"] != "0" {
		t.Fatalf("<video> do accept = %v", v.Attrs)
	}
	if _, has := v.Attrs["enc"]; has {
		t.Fatal("o <video> do accept não anuncia enc (o vp8 antigo deixou de existir)")
	}
}

func TestPreacceptWithVideoUsesOfferCapability(t *testing.T) {
	n := BuildPreacceptStanza(vPeer(), "CALL1", vCreator(), true)
	want := []string{"audio", "video", "encopt", "capability"}
	if got := childTagsOf(n, "preaccept"); !equalTags(got, want) {
		t.Fatalf("filhos do preaccept = %v, esperado %v", got, want)
	}
	v, _ := childOf(n, "preaccept", "video")
	if v.Attrs["dec"] != "H264" || v.Attrs["screen_width"] != "0" || v.Attrs["screen_height"] != "0" {
		t.Fatalf("<video> do preaccept = %v", v.Attrs)
	}
	if got := capabilityOf(t, n, "preaccept"); !bytes.Equal(got, wantOffer) {
		t.Fatalf("capability do preaccept com vídeo = %x, esperado %x (a do offer)", got, wantOffer)
	}
	audioOnly := BuildPreacceptStanza(vPeer(), "CALL1", vCreator(), false)
	if _, ok := childOf(audioOnly, "preaccept", "video"); ok {
		t.Fatal("preaccept de voz não pode ter <video>")
	}
}

func TestOfferHasVideo(t *testing.T) {
	with := waBinary.Node{Tag: "offer", Content: []waBinary.Node{{Tag: "audio"}, {Tag: "video"}, {Tag: "net"}}}
	without := waBinary.Node{Tag: "offer", Content: []waBinary.Node{{Tag: "audio"}, {Tag: "net"}}}
	if !OfferHasVideo(&with) || OfferHasVideo(&without) || OfferHasVideo(nil) {
		t.Fatal("OfferHasVideo errou")
	}
}

func intp(v int) *int { return &v }

func TestVideoStateShapes(t *testing.T) {
	up := BuildVideoStateWithParams(VideoStateParams{CallID: "C", To: vPeer(), CallCreator: vCreator(), WrapperID: "w",
		State: VideoStateUpgradeRequestV2, Dec: VideoDecRequest, DeviceOrientation: intp(0)})
	v, _ := childOf2(up)
	if v.Attrs["state"] != "11" || v.Attrs["dec"] != "H264" || v.Attrs["voip_settings"] != "video" || v.Attrs["device_orientation"] != "0" {
		t.Fatalf("pedido de upgrade = %v", v.Attrs)
	}
	acc := BuildVideoStateWithParams(VideoStateParams{CallID: "C", To: vPeer(), CallCreator: vCreator(), WrapperID: "w",
		State: VideoStateUpgradeAccept, Dec: VideoDecAccept})
	if v, _ := childOf2(acc); v.Attrs["state"] != "4" || v.Attrs["dec"] != "H264,AV1" {
		t.Fatalf("aceite = %v", v.Attrs)
	}
	on := BuildVideoStateWithParams(VideoStateParams{CallID: "C", To: vPeer(), CallCreator: vCreator(), WrapperID: "w", State: VideoStateEnabled})
	if v, _ := childOf2(on); v.Attrs["state"] != "1" {
		t.Fatalf("ligado = %v", v.Attrs)
	} else if _, has := v.Attrs["dec"]; has {
		t.Fatal("o estado 1 não leva dec")
	}
	stop := BuildVideoStateWithParams(VideoStateParams{CallID: "C", To: vPeer(), CallCreator: vCreator(), WrapperID: "w",
		State: VideoStateStopped, DeviceOrientation: intp(0)})
	if v, _ := childOf2(stop); v.Attrs["state"] != "6" || v.Attrs["device_orientation"] != "0" {
		t.Fatalf("parar = %v", v.Attrs)
	}
}

func childOf2(n waBinary.Node) (waBinary.Node, bool) {
	kids, _ := n.Content.([]waBinary.Node)
	if len(kids) == 0 {
		return waBinary.Node{}, false
	}
	return kids[0], true
}

// Sem o ack tipado o outro lado cancela a mudança de vídeo em ~5 s; o ack genérico do whatsmeow não serve.
func TestVideoAckIsTypedAndKeepsCompanionRouting(t *testing.T) {
	from := vPeer()
	participant := types.NewJID("333333333333333", types.HiddenUserServer)
	recipient := types.NewJID("444444444444444", types.HiddenUserServer)
	orig := &waBinary.Node{Tag: "call", Attrs: waBinary.Attrs{"id": "wrap", "from": from, "participant": participant, "recipient": recipient}}
	ack, ok := BuildVideoAck(orig)
	if !ok {
		t.Fatal("stanza roteável rejeitado")
	}
	if ack.Tag != "ack" || ack.Attrs["class"] != "call" || ack.Attrs["type"] != "video" || ack.Attrs["id"] != "wrap" {
		t.Fatalf("ack = %+v", ack.Attrs)
	}
	if ack.Attrs["participant"] != participant || ack.Attrs["recipient"] != recipient {
		t.Fatalf("roteamento de aparelho companheiro perdido: %+v", ack.Attrs)
	}
	if _, ok := BuildVideoAck(nil); ok {
		t.Fatal("nil")
	}
	if _, ok := BuildVideoAck(&waBinary.Node{Tag: "call", Attrs: waBinary.Attrs{}}); ok {
		t.Fatal("sem id/from não é roteável")
	}
}
