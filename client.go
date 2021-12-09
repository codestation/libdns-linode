package linode

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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
		record := libdns.Record{
			Name:  entry.Name,
			Value: entry.Target,
			Type:  string(entry.Type),
			TTL:   time.Duration(entry.TTLSec) * time.Second,
			ID:    strconv.Itoa(entry.ID),
		}
		records = append(records, record)
	}

	return records, nil
}

func (p *Provider) addOrUpdateDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	if record.ID == "" {
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

	createOpts := linodego.DomainRecordCreateOptions{
		Name:   record.Name,
		Target: record.Value,
		Type:   linodego.DomainRecordType(record.Type),
		TTLSec: int(record.TTL.Seconds()),
	}

	rec, err := p.client.CreateDomainRecord(ctx, domainID, createOpts)
	if err != nil {
		return record, err
	}
	record.ID = strconv.Itoa(rec.ID)

	return record, nil
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

	recordID, err := strconv.Atoi(record.ID)
	if err != nil {
		return record, err
	}

	updateOpts := linodego.DomainRecordUpdateOptions{
		Name:   strings.Trim(strings.ReplaceAll(record.Name, zone, ""), "."),
		Target: record.Value,
		Type:   linodego.DomainRecordType(record.Type),
		TTLSec: int(record.TTL.Seconds()),
	}

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

	id, err := strconv.Atoi(record.ID)
	if err != nil {
		return record, err
	}

	err = p.client.DeleteDomainRecord(ctx, domainID, id)
	if err != nil {
		return record, err
	}

	return record, nil
}
