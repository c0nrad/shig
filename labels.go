package main

import (
	"bytes"
	"errors"
)

type LabelManager struct {
	KnownLabels map[uint16][]byte
}

func NewLabelManager() LabelManager {
	return LabelManager{KnownLabels: make(map[uint16][]byte)}
}

func (m *LabelManager) Resolve(r *bytes.Reader) ([]byte, error) {
	if r.Len() == 0 {
		panic("Not a valid label")
	}

	startOffset := uint16(int(r.Size()) - r.Len())

	out := []byte{}
	for {
		l, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		if l == 0 {
			break
		}

		// is pointer
		if l&0xC0 == 0xC0 {
			l = l & 0x3F

			s, err := r.ReadByte()

			if err != nil {
				return nil, err
			}

			offset := uint16(l)<<8 | uint16(s)
			// fmt.Println("requesting offset", offset)

			label, ok := m.KnownLabels[offset]
			if !ok {
				return nil, errors.New("no label exists at offset")
			}

			out = append(out, label...)
			break
		}

		currOffset := uint16(int(r.Size())-r.Len()) - 1 //for size

		dater := make([]byte, l)
		n, err := r.Read(dater)
		if byte(n) != l {
			return nil, errors.New("didn't read full label")
		}

		if err != nil {
			return nil, err
		}

		out = append(out, dater...)
		out = append(out, byte('.'))

		// fmt.Println("Writing", currOffset, string(dater))
		m.KnownLabels[currOffset] = dater

	}

	// fmt.Println("Writing", startOffset, string(out))
	m.KnownLabels[startOffset] = out

	return out, nil
}
