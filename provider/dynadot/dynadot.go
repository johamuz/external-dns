package dynadot

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "strings"

    "github.com/kubernetes-sigs/external-dns/endpoint"
    "github.com/kubernetes-sigs/external-dns/provider"
)

// DynadotProvider is an implementation of Provider for Dynadot.
type DynadotProvider struct {
    client       *DynadotClient
    domainFilter endpoint.DomainFilter
    dryRun       bool
}

// NewDynadotProvider initializes a new Dynadot Provider.
func NewDynadotProvider(domainFilter endpoint.DomainFilter, dryRun bool, apiKey string) (*DynadotProvider, error) {
    if apiKey == "" {
        return nil, errors.New("missing API key for Dynadot")
    }

    client := NewDynadotClient(http.DefaultClient, apiKey)
    return &DynadotProvider{
        client:       client,
        domainFilter: domainFilter,
        dryRun:       dryRun,
    }, nil
}

// Records returns the list of records in all relevant zones.
func (p *DynadotProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
    zones, err := p.client.ListDomains()
    if err != nil {
        return nil, fmt.Errorf("Dynadot: failed to list domains: %w", err)
    }

    var endpoints []*endpoint.Endpoint
    for _, zone := range zones {
        if p.domainFilter.Match(zone.Name) {
            records, err := p.client.ListRecords(zone.Name)
            if err != nil {
                return nil, fmt.Errorf("Dynadot: failed to list records for domain %s: %w", zone.Name, err)
            }

            for _, record := range records {
                endpoints = append(endpoints, endpoint.NewEndpoint(record.Name, record.Type, record.Content))
            }
        }
    }
    return endpoints, nil
}

// ApplyChanges applies a given set of changes in a given zone.
func (p *DynadotProvider) ApplyChanges(ctx context.Context, changes *provider.Changes) error {
    if p.dryRun {
        for _, ep := range changes.Create {
            fmt.Printf("Would create record: %s %s %s\n", ep.DNSName, ep.RecordType, ep.Targets)
        }
        for _, ep := range changes.UpdateOld {
            fmt.Printf("Would update record: %s %s %s\n", ep.DNSName, ep.RecordType, ep.Targets)
        }
        for _, ep := range changes.Delete {
            fmt.Printf("Would delete record: %s %s\n", ep.DNSName, ep.RecordType)
        }
        return nil
    }

    for _, ep := range changes.Create {
        err := p.createRecord(ctx, ep)
        if err != nil {
            return err
        }
    }

    for _, ep := range changes.UpdateOld {
        err := p.deleteRecord(ctx, ep)
        if err != nil {
            return err
        }
    }

    for _, ep := range changes.UpdateNew {
        err := p.createRecord(ctx, ep)
        if err != nil {
            return err
        }
    }

    for _, ep := range changes.Delete {
        err := p.deleteRecord(ctx, ep)
        if err != nil {
            return err
        }
    }

    return nil
}

func (p *DynadotProvider) createRecord(ctx context.Context, ep *endpoint.Endpoint) error {
    zoneName := p.extractZone(ep.DNSName)
    record := DynadotRecord{
        Name:    ep.DNSName,
        Type:    ep.RecordType,
        Content: strings.Join(ep.Targets, ","),
        TTL:     300,
    }

    _, err := p.client.CreateRecord(zoneName, record)
    if err != nil {
        return fmt.Errorf("Dynadot: failed to create record %s: %w", ep.DNSName, err)
    }
    return nil
}

func (p *DynadotProvider) deleteRecord(ctx context.Context, ep *endpoint.Endpoint) error {
    zoneName := p.extractZone(ep.DNSName)
    recordID, err := p.findRecordID(ctx, zoneName, ep)
    if err != nil {
        return fmt.Errorf("Dynadot: failed to find record ID for %s: %w", ep.DNSName, err)
    }

    err = p.client.DeleteRecord(zoneName, recordID)
    if err != nil {
        return fmt.Errorf("Dynadot: failed to delete record %s: %w", ep.DNSName, err)
    }
    return nil
}

func (p *DynadotProvider) findRecordID(ctx context.Context, zoneName string, ep *endpoint.Endpoint) (string, error) {
    records, err := p.client.ListRecords(zoneName)
    if err != nil {
        return "", fmt.Errorf("Dynadot: failed to list records for domain %s: %w", zoneName, err)
    }

    for _, record := range records {
        if record.Name == ep.DNSName && record.Type == ep.RecordType && record.Content == strings.Join(ep.Targets, ",") {
            return record.ID, nil
        }
    }
    return "", fmt.Errorf("Dynadot: record not found")
}

func (p *DynadotProvider) extractZone(dnsName string) string {
    parts := strings.Split(dnsName, ".")
    if len(parts) > 1 {
        return strings.Join(parts[len(parts)-2:], ".")
    }
    return dnsName
}
