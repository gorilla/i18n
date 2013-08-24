// Copyright 2013 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gettext

import (
	"fmt"
	"io"
	"net/textproto"
	"sort"
)

// TODO: plural rules interface
// TODO: catalog methods: ContextSingular(), Plural(), ContextPlural()...

// ----------------------------------------------------------------------------

// NewCatalog returns a new catalog instance.
func NewCatalog() *Catalog {
	return &Catalog{
		msgs: map[string]*Message{},
	}
}

// Catalog stores translations.
type Catalog struct {
	Header textproto.MIMEHeader
	msgs   map[string]*Message
	keys   []string
}

// Singular returns a singular string stored in the catalog, optionally
// formatting it using the provided arguments.
func (c *Catalog) Singular(key string, args ...interface{}) string {
	if msg, ok := c.msgs[key]; ok {
		if text := msg.Str; text != nil {
			if len(args) == 0 {
				return string(text)
			}
			return fmt.Sprintf(string(text), args...)
		}
	}
	return key
}

// ReadMo reads a MO file from r and adds its messages to the catalog.
func (c *Catalog) ReadMo(r io.ReadSeeker) error {
	iter := ReadMo(r)
	size := iter.Size()
	for i := 0; i < size; i++ {
		msg, err := iter.Next()
		if err != nil {
			return err
		}
		// TODO: check this error
		c.setMessage(msg)
	}
	return nil
}

func (c *Catalog) setMessage(msg *Message) error {
	key, err := c.key(msg.Ctxt, msg.Id)
	if err != nil {
		return err
	}
	if _, ok := c.msgs[key]; ok {
		return fmt.Errorf("Message key already exists: %q.", key)
	}
	if len(key) == 0 {
		if c.Header != nil {
			return fmt.Errorf("Catalog header already exists.")
		}
		c.Header = bytesToHeader(msg.Str)
	}
	c.msgs[key] = msg
	c.keys = append(c.keys, key)
	return nil
}

func (c *Catalog) key(ctxt, id []byte) (string, error) {
	if id == nil {
		return "", fmt.Errorf("Invalid msgid.")
	}
	if ctxt == nil {
		return string(id), nil
	}
	return fmt.Sprintf("%s%s%s", ctxt, string('\x04'), id), nil
}

// Iter returns a messages iterator for this catalog.
func (c *Catalog) Iter() Iterator {
	// Note: as it is, new messages can't be added to the catalog when using
	// the iterator, because it would result in unsorted keys.
	sort.Strings(c.keys)
	return &catalogIterator{ctg: c}
}

// ----------------------------------------------------------------------------

// catalogIterator iterates over the messages stored in a catalog.
type catalogIterator struct {
	ctg *Catalog
	pos int
}

// Size returns the amount of messages provided by the iterator.
func (i *catalogIterator) Size() int {
	return len(i.ctg.keys)
}

// Next returns the next message. At the end of the iteration,
// io.EOF is returned as the error.
func (i *catalogIterator) Next() (*Message, error) {
	if i.pos < len(i.ctg.keys) {
		key := i.ctg.keys[i.pos]
		msg := i.ctg.msgs[key]
		if len(key) == 0 {
			msg.Str = headerToBytes(i.ctg.Header)
		}
		i.pos += 1
		return msg, nil
	}
	return nil, io.EOF
}
