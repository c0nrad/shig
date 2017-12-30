package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestHeaderSerialize(t *testing.T) {
	h := Header{
		ID:      0xAAAA,
		QR:      0,
		Opcode:  0,
		TC:      0,
		RD:      1,
		QDCOUNT: 1,
	}

	out, err := h.Serialize()
	if err != nil {
		t.Error(err)
	}

	correct := []byte{0xAA, 0xAA, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	if bytes.Compare(out, correct) != 0 {
		fmt.Println(out)
		fmt.Println(correct)
		t.Error("Failed to serialize header")
	}
}

func TestQuestionSerialize(t *testing.T) {
	q := Question{
		QNAME:  []byte("example.com"),
		QTYPE:  1,
		QCLASS: 1,
	}

	out, err := q.Serialize()
	if err != nil {
		t.Error(err)
	}

	correct := []byte{0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0x01, 0x00, 0x01}

	if bytes.Compare(out, correct) != 0 {
		fmt.Println(out)
		fmt.Println(correct)
		t.Error("Failed to serialize question")
	}
}

func TestHeaderDeserialize(t *testing.T) {
	in := []byte{170, 170, 129, 128, 0, 1, 0, 1, 0, 0, 0, 0}

	var header Header

	r := bytes.NewReader(in)
	err := header.Deserialize(r)

	if err != nil {
		t.Error(err)
	}

	if header.ID != 0xAAAA {
		t.Error("Failed to deserialize id")
	}

	if header.QR != 1 {
		t.Error("Failed to deserialize qr")
	}

	if header.AA != 0 {
		t.Error("Failed to deserialize aa")
	}

	if header.RD != 1 {
		t.Error("Failed to deserialize rd")
	}

	if header.RA != 1 {
		t.Error("Failed to deserialize ra")
	}

	if header.RCODE != 0 {
		t.Error("Failed to deserialize rcode")
	}

	if header.ARCOUNT != 0 {
		t.Error("Failed to deserialize arcount")
	}

	if header.ANCOUNT != 1 {
		t.Error("Failed to deserialize ancount")
	}

	if header.QDCOUNT != 1 {
		t.Error("Failed to deserialize qcount")
	}

	if header.NSCOUNT != 0 {
		t.Error("Failed to deserialize nscount")
	}

	reserialize, err := header.Serialize()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(reserialize, in) {
		t.Error("Failed to reserialize byte steam")
	}
}

func TestQuestionDeserialize(t *testing.T) {
	in := []byte{0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0x01, 0x00, 0x01}

	var question Question
	r := bytes.NewReader(in)

	l := NewLabelManager()

	err := question.Deserialize(r, &l)

	if err != nil {
		t.Error(err)
	}

	reserialize, err := question.Serialize()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(reserialize, in) {
		fmt.Println(reserialize)
		fmt.Println(in)

		t.Error("Failed to reserialize byte steam")
	}
}

func TestResourceDeserialize(t *testing.T) {
	in := []byte{192, 12, 0, 1, 0, 1, 0, 0, 70, 56, 0, 4, 93, 184, 216, 34}

	var resource Resource
	r := bytes.NewReader(in)

	l := NewLabelManager()

	err := resource.Deserialize(r, &l)

	if err != nil {
		t.Error(err)
	}

	if resource.RDLENGTH != 4 {
		t.Error("Failed to deserialize RDLENGTH")
	}

	if !bytes.Equal(resource.RDATA, []byte{93, 184, 216, 34}) {
		t.Error("Failed to deserialize RDATA")

	}

}

//solution := []byte{170, 170, 129, 128, 0, 1, 0, 1, 0, 0, 0, 0, 7, 101, 120, 97, 109, 112, 108, 101, 3, 99, 111, 109, 0, 0, 1, 0, 1, 192, 12, 0, 1, 0, 1, 0, 0, 70, 56, 0, 4, 93, 184, 216, 34}
