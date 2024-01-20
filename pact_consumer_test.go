package pactissue

import (
	context "context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	message "github.com/pact-foundation/pact-go/v2/message/v4"
	"github.com/stretchr/testify/require"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TestFunc func(*grpc.ClientConn) error

func runTest(t *testing.T, grpcInteraction string, testFunc TestFunc) {
	p, _ := message.NewSynchronousPact(message.Config{
		Consumer: "sample-consumer",
		Provider: "sample-provider",
	})

	err := p.AddSynchronousMessage("Sample").
		UsingPlugin(message.PluginConfig{
			Plugin: "protobuf",
		}).
		WithContents(grpcInteraction, "application/protobuf").
		StartTransport("grpc", "127.0.0.1", nil). // For plugin tests, we can't assume if a transport is needed, so this is optional
		ExecuteTest(t, func(transport message.TransportConfig, m message.SynchronousMessage) error {
			fmt.Println("gRPC transport running on", transport)
			conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", transport.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer conn.Close()

			return testFunc(conn)
		})

	require.NoError(t, err)
}

func TestBrokenProto(t *testing.T) {
	protoPath, err := filepath.Abs("sample.proto")
	require.NoError(t, err)

	grpcInteraction := `{
		"pact:proto": "` + protoPath + `",
		"pact:proto-service": "BrokenSampleService/GetSample",
		"pact:content-type": "application/protobuf",
		"request": {
			"type": [
				"notEmpty('TYPE1')"
			]
		},
		"response": {
			"ok": "matching(boolean, true)"
		}
	}`

	testFunc := func(conn *grpc.ClientConn) error {
		// Create the gRPC client
		c := NewBrokenSampleServiceClient(conn)
		req := &BrokenSampleRequest{
			Type: []BrokenSampleRequest_Type{BrokenSampleRequest_TYPE1},
		}

		// Now we can make a normal gRPC request
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := c.GetSample(ctx, req)

		if err != nil {
			t.Fatal(err.Error())
		}

		require.True(t, resp.GetOk())

		return nil
	}
	runTest(t, grpcInteraction, testFunc)
}

func TestWorkingProto(t *testing.T) {
	protoPath, err := filepath.Abs("sample.proto")
	require.NoError(t, err)

	grpcInteraction := `{
		"pact:proto": "` + protoPath + `",
		"pact:proto-service": "WorkingSampleService/GetSample",
		"pact:content-type": "application/protobuf",
		"request": {
			"type": "notEmpty('TYPE1')"
		},
		"response": {
			"ok": "matching(boolean, true)"
		}
	}`

	testFunc := func(conn *grpc.ClientConn) error {
		// Create the gRPC client
		c := NewWorkingSampleServiceClient(conn)
		req := &WorkingSampleRequest{
			Type: WorkingSampleRequest_TYPE1,
		}

		// Now we can make a normal gRPC request
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := c.GetSample(ctx, req)

		if err != nil {
			t.Fatal(err.Error())
		}

		require.True(t, resp.GetOk())

		return nil
	}

	runTest(t, grpcInteraction, testFunc)
}
