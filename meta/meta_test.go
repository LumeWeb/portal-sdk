package meta

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalmeta "go.lumeweb.com/portal-sdk/internal/meta"
	metamocks "go.lumeweb.com/portal-sdk/internal/meta/mocks"
)

// newMockMetaClient returns a service under test wired to a mock meta client.
func newMockMetaClient(t *testing.T) (*MetaClient, *metamocks.MockClientWithResponsesInterface) {
	mockClient := metamocks.NewMockClientWithResponsesInterface(t)
	cid := &CIDService{client: mockClient}
	stats := &StatsService{client: mockClient}
	return &MetaClient{client: mockClient, cid: cid, stats: stats}, mockClient
}

func TestMetaClient_Constructor(t *testing.T) {
	t.Run("NewClient requires a valid endpoint", func(t *testing.T) {
		_, err := NewClient()
		require.NoError(t, err)
	})
}

func TestMetaClient_CIDAndStats(t *testing.T) {
	t.Run("returns accessors for CID and Stats services", func(t *testing.T) {
		c, _ := newMockMetaClient(t)
		assert.NotNil(t, c.CID())
		assert.NotNil(t, c.Stats())
	})
}

func TestCIDService_GetDAG(t *testing.T) {
	t.Run("returns DAG export on success", func(t *testing.T) {
		c, mockClient := newMockMetaClient(t)

		expected := &internalmeta.DAGExportResponse{
			RootCid:        "bafyroothash",
			TotalBlocks:    3,
			TotalSizeBytes: 4096,
			Blocks: []internalmeta.DAGBlock{
				{Cid: "bafyblock1", Size: 1024, Links: []internalmeta.DAGLink{{Cid: "bafylink1", Index: 0}}},
			},
		}

		mockClient.EXPECT().
			GetApiExportCidCidDagWithResponse(mock.Anything, "bafyroothash").
			Return(&internalmeta.GetApiExportCidCidDagResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expected,
			}, nil).
			Once()

		result, err := c.CID().GetDAG(context.Background(), "bafyroothash")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expected.RootCid, result.RootCid)
		assert.Equal(t, expected.TotalBlocks, result.TotalBlocks)
		assert.Len(t, result.Blocks, 1)
		assert.Equal(t, "bafyblock1", result.Blocks[0].Cid)
	})

	t.Run("returns error on HTTP failure", func(t *testing.T) {
		c, mockClient := newMockMetaClient(t)

		mockClient.EXPECT().
			GetApiExportCidCidDagWithResponse(mock.Anything, "bafyroothash").
			Return(nil, http.ErrAbortHandler).
			Once()

		result, err := c.CID().GetDAG(context.Background(), "bafyroothash")
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when response body is nil on 200", func(t *testing.T) {
		c, mockClient := newMockMetaClient(t)

		mockClient.EXPECT().
			GetApiExportCidCidDagWithResponse(mock.Anything, "bafyroothash").
			Return(&internalmeta.GetApiExportCidCidDagResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      nil,
			}, nil).
			Once()

		result, err := c.CID().GetDAG(context.Background(), "bafyroothash")
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error on forbidden status", func(t *testing.T) {
		c, mockClient := newMockMetaClient(t)

		mockClient.EXPECT().
			GetApiExportCidCidDagWithResponse(mock.Anything, "bafyroothash").
			Return(&internalmeta.GetApiExportCidCidDagResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusForbidden},
				Body:         []byte(`{"error":{"reason":"FORBIDDEN","details":"insufficient permissions"}}`),
			}, nil).
			Once()

		result, err := c.CID().GetDAG(context.Background(), "bafyroothash")
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCIDService_GetSiaObject(t *testing.T) {
	t.Run("returns Sia object on success", func(t *testing.T) {
		c, mockClient := newMockMetaClient(t)

		expected := &internalmeta.CIDExportResponse{
			Cid:       "bafysiaobject",
			CreatedAt: "2026-08-04T00:00:00Z",
			SizeBytes: 1024,
			SharedObject: internalmeta.SharedObject{
				DataKey: []int{1, 2, 3},
				Slabs: []internalmeta.SlabSlice{
					{Length: 512, MinShards: 2, Offset: 0},
				},
			},
		}

		mockClient.EXPECT().
			GetApiExportCidCidSiaObjectWithResponse(mock.Anything, "bafysiaobject").
			Return(&internalmeta.GetApiExportCidCidSiaObjectResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expected,
			}, nil).
			Once()

		result, err := c.CID().GetSiaObject(context.Background(), "bafysiaobject")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expected.Cid, result.Cid)
		assert.Equal(t, len(expected.SharedObject.Slabs), 1)
		assert.Equal(t, expected.SharedObject.Slabs[0].Length, result.SharedObject.Slabs[0].Length)
	})

	t.Run("returns error on HTTP failure", func(t *testing.T) {
		c, mockClient := newMockMetaClient(t)

		mockClient.EXPECT().
			GetApiExportCidCidSiaObjectWithResponse(mock.Anything, "bafysiaobject").
			Return(nil, http.ErrAbortHandler).
			Once()

		result, err := c.CID().GetSiaObject(context.Background(), "bafysiaobject")
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCIDService_GetStats(t *testing.T) {
	t.Run("returns CID stats on success", func(t *testing.T) {
		c, mockClient := newMockMetaClient(t)

		pinned := true
		expected := &internalmeta.CIDStatsResponse{
			Cid:       "bafyroothash",
			Pinned:    pinned,
			PinCount:  5,
			SizeBytes: 4096,
		}

		mockClient.EXPECT().
			GetApiStatsCidCidWithResponse(mock.Anything, "bafyroothash").
			Return(&internalmeta.GetApiStatsCidCidResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expected,
			}, nil).
			Once()

		result, err := c.CID().GetStats(context.Background(), "bafyroothash")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expected.Cid, result.Cid)
		assert.Equal(t, expected.PinCount, result.PinCount)
	})

	t.Run("returns error on not found status", func(t *testing.T) {
		c, mockClient := newMockMetaClient(t)

		mockClient.EXPECT().
			GetApiStatsCidCidWithResponse(mock.Anything, "bafyroothash").
			Return(&internalmeta.GetApiStatsCidCidResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusNotFound},
				Body:         []byte(`{"error":{"reason":"NOT_FOUND","details":"CID not found"}}`),
			}, nil).
			Once()

		result, err := c.CID().GetStats(context.Background(), "bafyroothash")
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestStatsService_GetAggregate(t *testing.T) {
	t.Run("returns aggregate stats on success", func(t *testing.T) {
		c, mockClient := newMockMetaClient(t)

		expected := &internalmeta.AggregateStatsResponse{
			TotalPins:         100,
			TotalStorageBytes: 123456,
			TotalUploads:      42,
		}

		mockClient.EXPECT().
			GetApiStatsAggregateWithResponse(mock.Anything).
			Return(&internalmeta.GetApiStatsAggregateResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expected,
			}, nil).
			Once()

		result, err := c.Stats().GetAggregate(context.Background())
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expected.TotalPins, result.TotalPins)
		assert.Equal(t, expected.TotalStorageBytes, result.TotalStorageBytes)
	})
}

func TestStatsService_GetProtocols(t *testing.T) {
	t.Run("returns protocol stats on success", func(t *testing.T) {
		c, mockClient := newMockMetaClient(t)

		expected := &internalmeta.ProtocolStatsResponse{
			Protocols: []internalmeta.ProtocolStat{
				{Protocol: "ipfs", TotalPins: 10, TotalStorageBytes: 2048, TotalUploads: 3},
			},
		}

		mockClient.EXPECT().
			GetApiStatsProtocolsWithResponse(mock.Anything).
			Return(&internalmeta.GetApiStatsProtocolsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      expected,
			}, nil).
			Once()

		result, err := c.Stats().GetProtocols(context.Background())
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Protocols, 1)
		assert.Equal(t, "ipfs", result.Protocols[0].Protocol)
	})
}
