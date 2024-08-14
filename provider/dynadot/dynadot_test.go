func TestDynadotProvider(t *testing.T) {
	mockClient := NewMockDynadotClient()
	provider := &DynadotProvider{
		client:       mockClient,
		domainFilter: endpoint.NewDomainFilter([]string{"example.com"}),
		dryRun:       false,
	}

	t.Run("Returns correct records", func(t *testing.T) {
		mockClient.domains["example.com"] = []DynadotRecord{
			{ID: "1", Name: "test.example.com", Type: "A", Content: "1.2.3.4"},
		}

		records, err := provider.Records(context.Background())
		require.NoError(t, err)
		require.Len(t, records, 1)

		assert.Equal(t, "test.example.com", records[0].DNSName)
		assert.Equal(t, "A", records[0].RecordType)
		assert.Equal(t, "1.2.3.4", records[0].Targets[0])
	})

	t.Run("Creates a new record", func(t *testing.T) {
		ep := &endpoint.Endpoint{
			DNSName:    "new.example.com",
			RecordType: "A",
			Targets:    endpoint.Targets{"5.6.7.8"},
		}

		changes := &provider.Changes{
			Create: []*endpoint.Endpoint{ep},
		}

		err := provider.ApplyChanges(context.Background(), changes)
		require.NoError(t, err)

		records, err := provider.Records(context.Background())
		require.NoError(t, err)
		require.Len(t, records, 2)

		assert.Equal(t, "new.example.com", records[1].DNSName)
		assert.Equal(t, "A", records[1].RecordType)
		assert.Equal(t, "5.6.7.8", records[1].Targets[0])
	})

	t.Run("Deletes an existing record", func(t *testing.T) {
		ep := &endpoint.Endpoint{
			DNSName:    "test.example.com",
			RecordType: "A",
			Targets:    endpoint.Targets{"1.2.3.4"},
		}

		changes := &provider.Changes{
			Delete: []*endpoint.Endpoint{ep},
		}

		err := provider.ApplyChanges(context.Background(), changes)
		require.NoError(t, err)

		records, err := provider.Records(context.Background())
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.Equal(t, "new.example.com", records[0].DNSName)
	})

	t.Run("Updates an existing record", func(t *testing.T) {
		oldEp := &endpoint.Endpoint{
			DNSName:    "new.example.com",
			RecordType: "A",
			Targets:    endpoint.Targets{"5.6.7.8"},
		}

		newEp := &endpoint.Endpoint{
			DNSName:    "new.example.com",
			RecordType: "A",
			Targets:    endpoint.Targets{"9.10.11.12"},
		}

		changes := &provider.Changes{
			UpdateOld: []*endpoint.Endpoint{oldEp},
			UpdateNew: []*endpoint.Endpoint{newEp},
		}

		err := provider.ApplyChanges(context.Background(), changes)
		require.NoError(t, err)

		records, err := provider.Records(context.Background())
		require.NoError(t, err)
		require.Len(t, records, 1)

		assert.Equal(t, "new.example.com", records[0].DNSName)
		assert.Equal(t, "A", records[0].RecordType)
		assert.Equal(t, "9.10.11.12", records[0].Targets[0])
	})
}
ls