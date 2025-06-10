// Copyright 2025 codestation. All rights reserved.
// Use of this source code is governed by a MIT-license
// that can be found in the LICENSE file.

package linode

import (
	"fmt"
	"time"

	"github.com/libdns/libdns"
	"github.com/linode/linodego"
)

type Record struct {
	Record libdns.RR
	ID     int
}

func (d Record) RR() libdns.RR {
	return d.Record
}

func fromDomainRecord(entry linodego.DomainRecord) Record {
	return Record{
		Record: libdns.RR{
			Name: entry.Name,
			Data: entry.Target,
			Type: string(entry.Type),
			TTL:  time.Duration(entry.TTLSec) * time.Second,
		},
		ID: entry.ID,
	}
}

func FromLibdnsRecord(r libdns.Record, id int) Record {
	rr := r.RR()
	return Record{
		Record: rr,
		ID:     id,
	}
}

func asPointer[T any](v T) *T {
	return &v
}

func toDomainRecordCreate(record libdns.Record) linodego.DomainRecordCreateOptions {
	rr := record.RR()

	opts := linodego.DomainRecordCreateOptions{
		Type:   linodego.DomainRecordType(rr.Type),
		Name:   rr.Name,
		TTLSec: int(rr.TTL.Seconds()),
	}

	if rec, ok := rr.Parse(); ok == nil {
		switch r := rec.(type) {
		case libdns.MX:
			opts.Priority = asPointer(int(r.Preference))
			opts.Target = r.Target
		case libdns.SRV:
			opts.Priority = asPointer(int(r.Priority))
			opts.Target = r.Target
			opts.Port = asPointer(int(r.Port))
			opts.Weight = asPointer(int(r.Weight))
			opts.Protocol = asPointer(r.Transport)
			opts.Service = asPointer(r.Service)
		case libdns.CAA:
			opts.Target = r.Value
			opts.Tag = asPointer(r.Tag)
		default:
			opts.Target = rr.Data
		}
	}

	return opts
}

func toDomainRecordUpdate(record libdns.Record) linodego.DomainRecordUpdateOptions {
	rr := record.RR()

	opts := linodego.DomainRecordUpdateOptions{
		Type:   linodego.DomainRecordType(rr.Type),
		Name:   rr.Name,
		TTLSec: int(rr.TTL.Seconds()),
	}

	if rec, ok := rr.Parse(); ok == nil {
		switch r := rec.(type) {
		case libdns.MX:
			opts.Priority = asPointer(int(r.Preference))
			opts.Target = r.Target
		case libdns.SRV:
			opts.Priority = asPointer(int(r.Priority))
			opts.Target = r.Target
			opts.Port = asPointer(int(r.Port))
			opts.Weight = asPointer(int(r.Weight))
			opts.Protocol = asPointer(r.Transport)
			opts.Service = asPointer(r.Service)
		case libdns.CAA:
			opts.Target = r.Value
			opts.Tag = asPointer(r.Tag)
		default:
			opts.Target = rr.Data
		}
	}

	return opts
}

func getRecordId(r libdns.Record) (int, error) {
	var id int
	if vr, err := r.(Record); err {
		id = vr.ID
	}

	if id == 0 {
		return 0, fmt.Errorf("record has no ID: %v", r)
	}

	return id, nil
}
