// Copyright 2025 codestation. All rights reserved.
// Use of this source code is governed by a MIT-license
// that can be found in the LICENSE file.

package linode

import (
	"fmt"
	"net/netip"
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

func fromDomainRecord(entry linodego.DomainRecord) (Record, error) {
	r, err := ToDomainRecord(entry)
	if err != nil {
		return Record{}, err
	}

	return FromLibdnsRecord(r, entry.ID), nil
}

func FromLibdnsRecord(r libdns.Record, id int) Record {
	rr := r.RR()
	return Record{
		Record: rr,
		ID:     id,
	}
}

func nameForLibdns(linodeName string) string {
	if linodeName == "" {
		return "@"
	}

	return linodeName
}

func nameForLinode(libdnsName, zone string) string {
	if libdnsName == "@" {
		return zone
	}

	return libdnsName
}

func ToDomainRecord(r linodego.DomainRecord) (libdns.Record, error) {
	switch r.Type {
	case linodego.RecordTypeA:
		fallthrough
	case linodego.RecordTypeAAAA:
		addr, err := netip.ParseAddr(r.Target)
		if err != nil {
			return Record{}, fmt.Errorf("invalid IP address %s: %v", r.Target, err)
		}

		return libdns.Address{
			Name: nameForLibdns(r.Name),
			TTL:  time.Duration(r.TTLSec) * time.Second,
			IP:   addr,
		}, nil
	case linodego.RecordTypeNS:
		return libdns.NS{
			Name:   nameForLibdns(r.Name),
			TTL:    time.Duration(r.TTLSec) * time.Second,
			Target: r.Target,
		}, nil
	case linodego.RecordTypeMX:
		return libdns.MX{
			Name:       nameForLibdns(r.Name),
			TTL:        time.Duration(r.TTLSec) * time.Second,
			Preference: uint16(r.Priority),
			Target:     r.Target,
		}, nil
	case linodego.RecordTypeCNAME:
		return libdns.CNAME{
			Name:   nameForLibdns(r.Name),
			TTL:    time.Duration(r.TTLSec) * time.Second,
			Target: r.Target,
		}, nil
	case linodego.RecordTypeTXT:
		return libdns.TXT{
			Name: nameForLibdns(r.Name),
			TTL:  time.Duration(r.TTLSec) * time.Second,
			Text: r.Target,
		}, nil
	case linodego.RecordTypeSRV:
		return libdns.SRV{
			Service:   asValue(r.Service),
			Transport: asValue(r.Protocol),
			Name:      r.Name,
			TTL:       time.Duration(r.TTLSec) * time.Second,
			Priority:  uint16(r.Priority),
			Weight:    uint16(r.Weight),
			Port:      uint16(r.Port),
			Target:    r.Target,
		}, nil
	case linodego.RecordTypePTR:
		return libdns.RR{
			Type: "PTR",
			Name: nameForLibdns(r.Name),
			Data: r.Target,
			TTL:  time.Duration(r.TTLSec) * time.Second,
		}, nil
	case linodego.RecordTypeCAA:
		return libdns.CAA{
			Name:  r.Name,
			TTL:   time.Duration(r.TTLSec) * time.Second,
			Tag:   asValue(r.Tag),
			Value: r.Target,
		}, nil
	default:
		return libdns.RR{}, fmt.Errorf("unsupported record type %s", r.Type)
	}
}

func asPointer[T any](v T) *T {
	return &v
}

func asValue[T any](v *T) T {
	if v != nil {
		return *v
	}
	var empty T
	return empty
}

func toDomainRecordCreate(record libdns.Record, zone string) linodego.DomainRecordCreateOptions {
	rr := record.RR()

	opts := linodego.DomainRecordCreateOptions{
		Type:   linodego.DomainRecordType(rr.Type),
		TTLSec: int(rr.TTL.Seconds()),
	}

	if rec, ok := rr.Parse(); ok == nil {
		switch r := rec.(type) {
		case libdns.Address:
			opts.Name = nameForLinode(r.Name, zone)
			opts.Target = r.IP.String()
		case libdns.CNAME:
			opts.Name = r.Name
			opts.Target = nameForLinode(r.Target, zone)
		case libdns.TXT:
			opts.Name = nameForLinode(r.Name, zone)
			opts.Target = r.Text
		case libdns.CAA:
			opts.Name = r.Name
			opts.Target = nameForLinode(r.Value, zone)
			opts.Tag = asPointer(r.Tag)
		case libdns.MX:
			opts.Name = r.Name
			opts.Priority = asPointer(int(r.Preference))
			opts.Target = r.Target
		case libdns.SRV:
			opts.Name = r.Name
			opts.Priority = asPointer(int(r.Priority))
			opts.Target = nameForLinode(r.Target, zone)
			opts.Port = asPointer(int(r.Port))
			opts.Weight = asPointer(int(r.Weight))
			opts.Protocol = asPointer(r.Transport)
			opts.Service = asPointer(r.Service)
		default:
			opts.Name = rr.Name
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
