// Copyright 2025 codestation. All rights reserved.
// Use of this source code is governed by a MIT-license
// that can be found in the LICENSE file.

package linode

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/libdns/libdns"
	"github.com/linode/linodego"
	"golang.org/x/oauth2"
)

type Client struct {
	client *linodego.Client
	mutex  sync.Mutex
}

func (p *Provider) getClient() error {
	if p.client == nil {
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: p.APIToken})

		oauth2Client := &http.Client{
			Transport: &oauth2.Transport{
				Source: tokenSource,
			},
		}

		client := linodego.NewClient(oauth2Client)
		p.client = &client
	}

	return nil
}

func (p *Provider) getDomainID(ctx context.Context, zone string) (int, error) {
	data, err := json.Marshal(map[string]string{"domain": zone})
	if err != nil {
		return 0, err
	}

	listOpts := linodego.NewListOptions(1, string(data))
	domains, err := p.client.ListDomains(ctx, listOpts)
	if err != nil {
		return 0, err
	}

	if len(domains) == 0 {
		return 0, errors.New("domain not found")
	}

	return domains[0].ID, nil
}

func (p *Provider) getDNSEntries(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.getClient()
	if err != nil {
		return nil, err
	}

	domainID, err := p.getDomainID(ctx, zone)
	if err != nil {
		return nil, err
	}

	opts := &linodego.ListOptions{}
	var records []libdns.Record

	domains, err := p.client.ListDomainRecords(ctx, domainID, opts)
	if err != nil {
		return nil, err
	}

	for _, entry := range domains {
		record := fromDomainRecord(entry)
		records = append(records, record)
	}

	return records, nil
}

func (p *Provider) addOrUpdateDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	recordId, err := getRecordId(record)
	if err != nil {
		return record, err
	}

	if recordId == 0 {
		return p.addDNSEntry(ctx, zone, record)
	} else {
		return p.updateDNSEntry(ctx, p.unFQDN(zone), record)
	}
}

func (p *Provider) addDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.getClient()
	if err != nil {
		return record, err
	}

	domainID, err := p.getDomainID(ctx, zone)
	if err != nil {
		return record, err
	}

	createOpts := toDomainRecordCreate(record)
	rec, err := p.client.CreateDomainRecord(ctx, domainID, createOpts)
	if err != nil {
		return record, err
	}

	r := FromLibdnsRecord(record, rec.ID)

	return r, nil
}

func (p *Provider) updateDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.getClient()
	if err != nil {
		return record, err
	}

	domainID, err := p.getDomainID(ctx, zone)
	if err != nil {
		return record, err
	}

	recordID, err := getRecordId(record)
	if err != nil {
		return record, err
	}

	updateOpts := toDomainRecordUpdate(record)
	_, err = p.client.UpdateDomainRecord(ctx, domainID, recordID, updateOpts)
	if err != nil {
		return record, err
	}

	return record, nil
}

func (p *Provider) removeDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.getClient()
	if err != nil {
		return record, err
	}

	domainID, err := p.getDomainID(ctx, zone)
	if err != nil {
		return record, err
	}

	recordID, err := getRecordId(record)
	if err != nil {
		return record, err
	}

	err = p.client.DeleteDomainRecord(ctx, domainID, recordID)
	if err != nil {
		return record, err
	}

	return record, nil
}
