// Copyright 2013 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gettext

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/textproto"
)

// Message stores a gettext message.
type Message struct {
	Ctxt      []byte       // msgctxt: message context, if any
	Id        []byte       // msgid: untranslated singular string
	IdPlural  []byte       // msgid_plural: untranslated plural string
	Str       []byte       // msgstr: translated singular string
	StrPlural [][]byte     // msgstr[n]: translated plural strings
	Meta      *MessageMeta // message comments from PO file
}

// MessageMeta stores meta-data from a gettext message.
type MessageMeta struct {
	TranslatorComments [][]byte
	ExtractedComments  [][]byte
	References         [][]byte
	Flags              [][]byte
	PrevCtxt           []byte
	PrevId             []byte
	PrevIdPlural       []byte
}

// Iterator iterates over gettext messages.
type Iterator interface {
	// Size returns the amount of messages provided by the iterator.
	Size() int
	// Next returns the next message. At the end of the iteration,
	// io.EOF is returned as the error.
	Next() (*Message, error)
}

// ----------------------------------------------------------------------------

// ReadMo reads a MO file from r and returns a messages iterator.
func ReadMo(r io.ReadSeeker) Iterator {
	return &moReader{reader: r}
}

// WriteMo writes a MO file to w using the provided messages iterator.
func WriteMo(w io.WriteSeeker, iter Iterator) error {
	writer := &moWriter{
		writer: w,
		iter:   iter,
	}
	if err := writer.writeAll(); err != nil {
		return err
	}
	return nil
}

// ----------------------------------------------------------------------------

const (
	// Magic byte order identifiers.
	bigEndian    uint32 = 0xde120495
	littleEndian uint32 = 0x950412de
)

var (
	// Separators for msgctxt, msgid, msgstr and plural equivalents.
	nulBytes = []byte("\x00")
	eotBytes = []byte("\x04")
)

type moRevision struct {
	Major, Minor uint16
}

type moHeader struct {
	MsgCount, IdTableOffset, StrTableOffset, HashSize, HashOffset uint32
}

type moPosition struct {
	Size, Offset uint32
}

// moReader reads a MO file.
type moReader struct {
	reader io.ReadSeeker    // stream reader
	order  binary.ByteOrder // byte order of the stream
	offset int64            // relative offset of the stream
	header *moHeader        // parsed catalog header
	pos    uint32           // iterator position
	err    error            // iterator error
}

func (r *moReader) init() {
	if r.header == nil {
		h, err := r.readHeader()
		if err != nil {
			h = moHeader{}
			r.err = err
		}
		r.header = &h
	}
}

// Size returns the amount of messages provided by the iterator.
func (r *moReader) Size() int {
	r.init()
	return int(r.header.MsgCount)
}

// Next returns the next message. At the end of the iteration,
// io.EOF is returned as the error.
func (r *moReader) Next() (*Message, error) {
	r.init()
	if r.err != nil {
		return nil, r.err
	}
	if r.pos >= r.header.MsgCount {
		r.err = io.EOF
		return nil, r.err
	}
	msg := Message{}
	var err error
	// Read msgid and msgstr.
	if msg.Id, err = r.readMessage(r.header.IdTableOffset + r.pos*8); err != nil {
		r.err = err
		return nil, err
	}
	if msg.Str, err = r.readMessage(r.header.StrTableOffset + r.pos*8); err != nil {
		r.err = err
		return nil, err
	}
	// Is this a context message?
	if idx := bytes.Index(msg.Id, eotBytes); idx != -1 {
		msg.Ctxt = msg.Id[:idx]
		msg.Id = msg.Id[idx+1:]
	}
	// Is this a plural message?
	if idx := bytes.Index(msg.Id, nulBytes); idx != -1 {
		msg.IdPlural = msg.Id[idx+1:]
		msg.Id = msg.Id[:idx]
		msg.StrPlural = bytes.Split(msg.Str, nulBytes)
		msg.Str = nil
	}
	r.pos += 1
	return &msg, nil
}

// readHeader reads the MO file header.
func (r *moReader) readHeader() (moHeader, error) {
	h := moHeader{}
	if err := r.seek(0); err != nil {
		return h, err
	}
	// byte 0: byte order.
	var order uint32
	if err := binary.Read(r.reader, binary.LittleEndian, &order); err != nil {
		return h, err
	}
	switch order {
	case bigEndian:
		r.order = binary.BigEndian
	case littleEndian:
		r.order = binary.LittleEndian
	default:
		return h, errors.New("Unable to identify the byte order.")
	}
	// byte 4: major revision number.
	// byte 6: minor revision number (ignored).
	revision := moRevision{}
	if err := binary.Read(r.reader, r.order, &revision); err != nil {
		return h, err
	}
	// From spec: "A program seeing an unexpected major revision
	// number should stop reading the MO file entirely".
	if revision.Major != 0 && revision.Major != 1 {
		return h, errors.New("Unexpected major revision number.")
	}
	// byte 8:  number of messages.
	// byte 12: index of messages table.
	// byte 16: index of translations table.
	// byte 20: size of hashing table (ignored).
	// byte 24: offset of hashing table (ignored).
	if err := binary.Read(r.reader, r.order, &h); err != nil {
		return h, err
	}
	// Increment offset by the amount we have read.
	r.offset += 28
	return h, nil
}

// readMessage reads a message or translation at the given stream offset.
func (r *moReader) readMessage(tableOffset uint32) ([]byte, error) {
	// Get message length and position.
	if err := r.seek(int64(tableOffset)); err != nil {
		return nil, err
	}
	pos := moPosition{}
	if err := binary.Read(r.reader, r.order, &pos); err != nil {
		return nil, err
	}
	// Increment offset by the amount we have read.
	r.offset += 8
	// Get the message itself.
	if err := r.seek(int64(pos.Offset)); err != nil {
		return nil, err
	}
	bytes := make([]byte, pos.Size)
	if err := binary.Read(r.reader, r.order, bytes); err != nil {
		return nil, err
	}
	// Increment offset by the amount we have read.
	r.offset += int64(pos.Size)
	return bytes, nil
}

// seek seeks the underlying reader relatively to the position in which
// it was initially provided.
func (r *moReader) seek(offset int64) error {
	if _, err := r.reader.Seek(offset-r.offset, 1); err != nil {
		return err
	}
	r.offset = offset
	return nil
}

// ----------------------------------------------------------------------------

// moWriter writes a MO file.
type moWriter struct {
	writer io.WriteSeeker // stream writer
	iter   Iterator       // messages to write
	offset int64          // relative offset of the stream
}

// writeAll writes the whole MO file.
//
// Providing the catalog header is left to the catalog implementation.
func (w *moWriter) writeAll() error {
	msgCount := uint32(w.iter.Size())
	// Write header.
	h := moHeader{
		MsgCount:       msgCount,
		IdTableOffset:  28,
		StrTableOffset: msgCount*8 + 28,
	}
	if err := w.writeHeader(h); err != nil {
		return err
	}
	offset := msgCount*16 + 28
	for i := uint32(0); i < msgCount; i++ {
		msg, err := w.iter.Next()
		if err != nil {
			return err
		}
		// Write msgid.
		b := append(make([]byte, 0), msg.Id...)
		if msg.Ctxt != nil {
			b = append(b[:0], append(append(msg.Ctxt, eotBytes...), b...)...)
		}
		if msg.IdPlural != nil {
			b = append(append(b, nulBytes...), msg.IdPlural...)
		}
		if err = w.writeMessage(h.IdTableOffset+i*8, offset, b); err != nil {
			return err
		}
		offset += uint32(len(b) + 1) // +1 for the NUL char separator.
		// Write msgstr.
		if msg.IdPlural == nil {
			b = msg.Str
		} else {
			b = bytes.Join(msg.StrPlural, nulBytes)
		}
		if err = w.writeMessage(h.StrTableOffset+i*8, offset, b); err != nil {
			return err
		}
		offset += uint32(len(b) + 1) // +1 for the NUL char separator.
	}
	return nil
}

// writeHeader writes the MO file header.
func (w *moWriter) writeHeader(header moHeader) error {
	if err := w.seek(0); err != nil {
		return err
	}
	// byte 0: magic number.
	if err := binary.Write(w.writer, binary.LittleEndian, littleEndian); err != nil {
		return err
	}
	// byte 4: major+minor revision number.
	if err := binary.Write(w.writer, binary.LittleEndian, moRevision{}); err != nil {
		return err
	}
	// bytes 8-24: header values.
	if err := binary.Write(w.writer, binary.LittleEndian, header); err != nil {
		return err
	}
	// Increment offset by the amount we have written.
	w.offset += 28
	return nil
}

// writeMessage writes a message or translation at the given stream offsets.
func (w *moWriter) writeMessage(tableOffset, msgOffset uint32, bytes []byte) error {
	// Write message length and position.
	if err := w.seek(int64(tableOffset)); err != nil {
		return err
	}
	pos := moPosition{uint32(len(bytes)), msgOffset}
	if err := binary.Write(w.writer, binary.LittleEndian, pos); err != nil {
		return err
	}
	// Increment offset by the amount we have written.
	w.offset += 8
	// Write the message itself.
	if err := w.seek(int64(msgOffset)); err != nil {
		return err
	}
	bytes = append(bytes, nulBytes...)
	if err := binary.Write(w.writer, binary.LittleEndian, bytes); err != nil {
		return err
	}
	// Increment offset by the amount we have written.
	w.offset += int64(len(bytes))
	return nil
}

// seek seeks the underlying writer relatively to the position in which
// it was initially provided.
func (w *moWriter) seek(offset int64) error {
	if _, err := w.writer.Seek(offset-w.offset, 1); err != nil {
		return err
	}
	w.offset = offset
	return nil
}

// ----------------------------------------------------------------------------

// bytesToHeader converts the provided bytes into a translations header.
func bytesToHeader(header []byte) textproto.MIMEHeader {
	reader := bufio.NewReader(bytes.NewReader(header))
	h, err := textproto.NewReader(reader).ReadMIMEHeader()
	if err == io.EOF {
		return h
	}
	return textproto.MIMEHeader{}
}

// headerToBytes converts the provided translations header into bytes.
func headerToBytes(header textproto.MIMEHeader) []byte {
	if header != nil {
		b := new(bytes.Buffer)
		for key, values := range header {
			for _, value := range values {
				// TODO: should we escape key or value somehow?
				b.WriteString(fmt.Sprintf("%s: %s\n", key, value))
			}
		}
		return b.Bytes()
	}
	return nil
}

// TODO: make this public, and use it.
// multiError groups non-fatal errors occurred when reading or writing gettext
// files.
type multiError []error

func (m multiError) Error() string {
	s, n := "", 0
	for _, e := range m {
		if e != nil {
			if n == 0 {
				s = e.Error()
			}
			n++
		}
	}
	switch n {
	case 0:
		return "(0 errors)"
	case 1:
		return s
	case 2:
		return s + " (and 1 other error)"
	}
	return fmt.Sprintf("%s (and %d other errors)", s, n-1)
}
