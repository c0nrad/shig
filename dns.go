package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

var TypeMapping = map[uint16]string{
	1:   "A",
	2:   "NS",
	5:   "CNAME",
	6:   "SOA",
	11:  "PTR",
	15:  "MX",
	16:  "TXT",
	28:  "AAAA",
	252: "AXFR",
	255: "ANY",
}

var ClassMapping = map[uint16]string{
	1: "IN",
}

var OpcodeMapping = map[uint8]string{
	0: "QUERY",
	1: "IQUERY",
	2: "STATUS",
}

var RcodeMapping = map[uint8]string{
	0: "NOERROR",
	1: "FORMATERROR",
	2: "SERVER FAILURE",
	3: "NAME ERROR",
	4: "NOT IMPLEMENTED",
	5: "REFUSED",
}

type Message struct {
	Header
	Questions   []Question
	Answers     []Resource
	Authorities []Resource
	Additionals []Resource

	labelManager LabelManager
}

func (m Message) Serialize() ([]byte, error) {

	headerBytes, err := m.Header.Serialize()
	if err != nil {
		return nil, err
	}

	questionBytes, err := m.Questions[0].Serialize()
	if err != nil {
		return nil, err
	}

	return append(headerBytes, questionBytes...), nil
}

func (m Message) String() string {
	out := m.Header.String() + "\n"

	for i := uint16(0); i < m.Header.QDCOUNT; i++ {
		out += m.Questions[i].String()
	}
	out += "\nAnswers:\n"

	for i := uint16(0); i < m.Header.ANCOUNT; i++ {
		out += m.Answers[i].String()
	}

	for i := uint16(0); i < m.Header.NSCOUNT; i++ {
		out += m.Authorities[i].String()
	}

	for i := uint16(0); i < m.Header.ARCOUNT; i++ {
		out += m.Additionals[i].String()
	}

	return out
}

func (m *Message) Deserialize(in []byte) error {

	m.labelManager.KnownLabels = make(map[uint16][]byte)

	r := bytes.NewReader(in)

	err := m.Header.Deserialize(r)
	if err != nil {
		return err
	}

	m.Questions = make([]Question, m.Header.QDCOUNT)
	for i := uint16(0); i < m.Header.QDCOUNT; i++ {
		err = m.Questions[i].Deserialize(r, &m.labelManager)
		if err != nil {
			return err
		}
	}

	m.Answers = make([]Resource, m.Header.ANCOUNT)
	for i := uint16(0); i < m.Header.ANCOUNT; i++ {
		err = m.Answers[i].Deserialize(r, &m.labelManager)
		if err != nil {
			return err
		}
	}

	m.Authorities = make([]Resource, m.Header.NSCOUNT)
	for i := uint16(0); i < m.Header.NSCOUNT; i++ {
		err = m.Authorities[i].Deserialize(r, &m.labelManager)
		if err != nil {
			return err
		}
	}

	m.Additionals = make([]Resource, m.Header.ARCOUNT)
	for i := uint16(0); i < m.Header.ARCOUNT; i++ {
		err = m.Additionals[i].Deserialize(r, &m.labelManager)
		if err != nil {
			return err
		}
	}
	return nil
}

type Header struct {
	ID uint16

	//0  1  2  3  4  5  6  7  8  9  A  B  C  D  E  F
	//+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	//|QR|   Opcode  |AA|TC|RD|RA|   Z    |   RCODE   |
	//+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	QR             uint8 // Query or response
	Opcode         uint8 // Type of question (QUERY, Inverse Query, Server Status (2))
	AA, TC, RD, RA uint8 // Authoritative Answer, Truncated, Recursion Desired, Decursion Available
	Z              uint8 // Zero
	RCODE          uint8 // Response Code (None, format, server failure, name error, unknown query, refused)

	QDCOUNT, ANCOUNT, NSCOUNT, ARCOUNT uint16 // Question Desired Count, Answer Count, Name Server Count, Additional Record
}

func (h Header) String() string {
	out := fmt.Sprintf("Header:\n")
	out += fmt.Sprintf("  OpCode: %s, Response Code: %s, ID: %X\n", OpcodeMapping[h.Opcode], RcodeMapping[h.RCODE], h.ID)
	out += fmt.Sprintf("  Flags: IsQuery: %b, Authoritative: %b, Truncated: %b, Recursion Desired: %b, Recursion Available: %b\n", h.QR, h.AA, h.TC, h.RD, h.RA)

	out += fmt.Sprintf("  Response: Query: %d", h.QDCOUNT)
	out += fmt.Sprintf(" Answer: %d", h.ANCOUNT)
	out += fmt.Sprintf(" Authority: %d", h.NSCOUNT)
	out += fmt.Sprintf(" Additional: %d\n", h.ARCOUNT)

	return out
}

type Question struct {
	QNAME  []byte
	QTYPE  uint16
	QCLASS uint16
}

func (q Question) String() string {
	out := fmt.Sprintf("Question:\n")
	out += fmt.Sprintf("  Name: %q", q.QNAME)
	out += fmt.Sprintf(" Type: %s", TypeMapping[q.QTYPE])
	out += fmt.Sprintf(" Class: %s\n", ClassMapping[q.QCLASS])

	return out
}

type Resource struct {
	NAME     []byte
	TYPE     uint16
	CLASS    uint16
	TTL      uint32
	RDLENGTH uint16
	RDATA    []byte

	RDATAClean string
}

// ;; ANSWER SECTION:
// c0nrad.io.		1799	IN	A	162.243.11.58
// c0nrad.io.		1799	IN	NS	ns1.digitalocean.com.
// c0nrad.io.		1799	IN	NS	ns2.digitalocean.com.
// c0nrad.io.		1799	IN	NS	ns3.digitalocean.com.
// c0nrad.io.		1799	IN	SOA	ns1.digitalocean.com. hostmaster.c0nrad.io. 1502922459 10800 3600 604800 1800
// c0nrad.io.		1799	IN	MX	1 aspmx.l.google.com.
// c0nrad.io.		1799	IN	MX	5 alt1.aspmx.l.google.com.
// c0nrad.io.		1799	IN	MX	5 alt2.aspmx.l.google.com.
// c0nrad.io.		1799	IN	MX	10 aspmx2.googlemail.com.
// c0nrad.io.		1799	IN	MX	10 aspmx3.googlemail.com.
func (r Resource) String() string {

	out := fmt.Sprintf("  %s %d %s %s %s\n", r.NAME, r.TTL, ClassMapping[r.CLASS], TypeMapping[r.TYPE], r.RDATAClean)

	return out
}

func CleanRecordData(rtype uint16, r *bytes.Reader, labelManager *LabelManager) string {
	// A 1, AAAA 28
	if rtype == 1 || rtype == 28 {
		rdata := make([]byte, 4)
		err := binary.Read(r, binary.BigEndian, &rdata)
		if err != nil {
			panic(err)
		}
		return net.IP(rdata).String()
	}

	// NS
	if rtype == 2 {
		out, err := labelManager.Resolve(r)
		if err != nil {
			panic(err)
		}

		return string(out)
	}

	// MX 15
	if rtype == 15 {
		var prefernce uint16
		err := binary.Read(r, binary.BigEndian, &prefernce)
		if err != nil {
			panic(err)
		}

		out, err := labelManager.Resolve(r)
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf("%d %s", prefernce, out)
	}

	// SOA 6
	if rtype == 6 {
		MNAME, err := labelManager.Resolve(r)
		if err != nil {
			panic(err)
		}

		RNAME, err := labelManager.Resolve(r)
		if err != nil {
			panic(err)
		}

		var serial, refresh, retry, expire, minimum uint32
		err = binary.Read(r, binary.BigEndian, &serial)
		if err != nil {
			panic(err)
		}

		err = binary.Read(r, binary.BigEndian, &refresh)
		if err != nil {
			panic(err)
		}

		err = binary.Read(r, binary.BigEndian, &retry)
		if err != nil {
			panic(err)
		}

		err = binary.Read(r, binary.BigEndian, &expire)
		if err != nil {
			panic(err)
		}

		err = binary.Read(r, binary.BigEndian, &minimum)
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf("%s %s %d %d %d %d %d", MNAME, RNAME, serial, refresh, retry, expire, minimum)

	}

	// CNAME 5
	if rtype == 5 {
		out, err := labelManager.Resolve(r)
		if err != nil {
			panic(err)
		}

		return string(out)
	}

	return ""
}

func (resource *Resource) Deserialize(r *bytes.Reader, labelManager *LabelManager) error {
	var err error

	resource.NAME, err = labelManager.Resolve(r)

	err = binary.Read(r, binary.BigEndian, &resource.TYPE)
	if err != nil {
		return err
	}

	err = binary.Read(r, binary.BigEndian, &resource.CLASS)
	if err != nil {
		return err
	}

	err = binary.Read(r, binary.BigEndian, &resource.TTL)
	if err != nil {
		return err
	}

	err = binary.Read(r, binary.BigEndian, &resource.RDLENGTH)
	if err != nil {
		return err
	}

	resource.RDATA = make([]byte, resource.RDLENGTH)
	err = binary.Read(r, binary.BigEndian, &resource.RDATA)
	if err != nil {
		return err
	}

	r.Seek(-int64(resource.RDLENGTH), io.SeekCurrent)

	resource.RDATAClean = CleanRecordData(resource.TYPE, r, labelManager)

	return nil
}

func (q Question) Serialize() ([]byte, error) {
	out := new(bytes.Buffer)

	labels := bytes.Split(q.QNAME, []byte("."))
	for _, l := range labels {
		if len(l) == 0 {
			continue
		}
		err := out.WriteByte(byte(len(l)))
		if err != nil {
			return nil, err
		}
		_, err = out.Write(l)
		if err != nil {
			return nil, err
		}
	}
	err := out.WriteByte(byte(0))
	if err != nil {
		return nil, err
	}

	err = binary.Write(out, binary.BigEndian, []uint16{q.QTYPE, q.QCLASS})
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func (q *Question) Deserialize(r *bytes.Reader, labelManager *LabelManager) error {
	var err error

	if r.Len() < 5 {
		return errors.New("incorrect question size")
	}

	q.QNAME, err = labelManager.Resolve(r)

	err = binary.Read(r, binary.BigEndian, &q.QTYPE)
	if err != nil {
		return err
	}

	err = binary.Read(r, binary.BigEndian, &q.QCLASS)
	if err != nil {
		return err
	}

	return nil
}

func (h Header) Serialize() ([]byte, error) {
	out := new(bytes.Buffer)

	flags := uint16(h.RCODE) | uint16(h.Z)<<4
	flags |= uint16(h.AA)<<10 | uint16(h.TC)<<9 | uint16(h.RD)<<8 | uint16(h.RA)<<7
	flags |= uint16(h.Opcode) << 11
	flags |= uint16(h.QR) << 15

	message := []uint16{h.ID, flags, h.QDCOUNT, h.ANCOUNT, h.NSCOUNT, h.ARCOUNT}

	err := binary.Write(out, binary.BigEndian, message)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func (h *Header) Deserialize(r *bytes.Reader) error {
	if r.Len() < 12 {
		return errors.New("not enough bytes for header")
	}

	var flags uint16

	err := binary.Read(r, binary.BigEndian, &h.ID)
	if err != nil {
		return err
	}

	err = binary.Read(r, binary.BigEndian, &flags)
	if err != nil {
		return err
	}

	// 00000000 00001111
	h.RCODE = uint8(0x000F & flags)
	flags >>= 4

	// 00000000 00000111
	h.Z = uint8(0x0007 & flags)
	flags >>= 3

	// 00000000 00000111
	h.RA = uint8(0x0001 & flags)
	flags >>= 1

	// 00000000 00000111
	h.RD = uint8(0x0001 & flags)
	flags >>= 1

	// 00000000 00000111
	h.TC = uint8(0x0001 & flags)
	flags >>= 1

	// 00000000 00000111
	h.AA = uint8(0x0001 & flags)
	flags >>= 1

	// 00000000 00000111
	h.Opcode = uint8(0x000F & flags)
	flags >>= 4

	// 00000000 00000111
	h.QR = uint8(0x0001 & flags)

	err = binary.Read(r, binary.BigEndian, &h.QDCOUNT)
	if err != nil {
		return err
	}

	err = binary.Read(r, binary.BigEndian, &h.ANCOUNT)
	if err != nil {
		return err
	}

	err = binary.Read(r, binary.BigEndian, &h.NSCOUNT)
	if err != nil {
		return err
	}

	err = binary.Read(r, binary.BigEndian, &h.ARCOUNT)
	if err != nil {
		return err
	}

	return nil
}

func Query(domain string, rtype uint16, ns string) ([]byte, error) {
	h := Header{
		ID:      0xAAAA,
		QR:      0,
		Opcode:  0,
		TC:      0,
		RD:      1,
		QDCOUNT: 1,
	}

	q := Question{
		QNAME:  []byte(domain),
		QTYPE:  rtype,
		QCLASS: 1,
	}

	m := Message{Header: h, Questions: []Question{q}}

	request, err := m.Serialize()
	if err != nil {
		return nil, err
	}

	con, err := net.Dial("udp", ns+":53")
	if err != nil {
		return nil, err
	}

	_, err = con.Write(request)
	if err != nil {
		return nil, err
	}

	response := make([]byte, 4096)
	n, err := con.Read(response)

	return response[0:n], err
}
