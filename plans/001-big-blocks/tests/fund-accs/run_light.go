package fundaccounts

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/celestiaorg/celestia-node/nodebuilder/node"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/nodekit"
	"github.com/celestiaorg/test-infra/plans/001-big-blocks/tests/common"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func RunLightNode(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Minute*time.Duration(runenv.IntParam("execution-time")),
	)
	defer cancel()

	err := nodekit.SetLoggersLevel("DEBUG")
	if err != nil {
		return err
	}

	syncclient := initCtx.SyncClient
	netclient := network.NewClient(syncclient, runenv)

	netclient.MustWaitNetworkInitialized(ctx)

	config := network.Config{
		Network: "default",
		Enable:  true,
		Default: network.LinkShape{
			Latency:   time.Duration(runenv.IntParam("latency")),
			Bandwidth: common.GetBandwidthValue(runenv.StringParam("bandwidth")),
		},
		CallbackState: "network-configured",
		RoutingPolicy: network.AllowAll,
	}

	config.IPv4 = runenv.TestSubnet

	// using the assigned `GlobalSequencer` id per each of instance
	// to fill in the last 2 octects of the new IP address for the instance
	ipC := byte((initCtx.GlobalSeq >> 8) + 1)
	ipD := byte(initCtx.GlobalSeq)
	config.IPv4.IP = append(config.IPv4.IP[0:2:2], ipC, ipD)

	err = netclient.ConfigureNetwork(ctx, &config)
	if err != nil {
		return err
	}

	// we need to get the validator's ip in info for grpc connection for pfd & gsbn
	appNode, err := common.GetValidatorInfo(ctx, syncclient, runenv.IntParam("validator"), int(initCtx.GroupSeq))
	if err != nil {
		return err
	}

	bridgeNode, err := common.GetBridgeNode(
		ctx,
		syncclient,
		initCtx.GroupSeq,
		runenv.IntParam("bridge"),
	)
	if err != nil {
		return err
	}

	ndhome := fmt.Sprintf("/.celestia-light-%d", int(initCtx.GlobalSeq))
	runenv.RecordMessage(ndhome)
	ip, err := initCtx.NetClient.GetDataNetworkIP()
	if err != nil {
		return err
	}

	trustedPeers := []string{bridgeNode.Maddr}
	cfg := nodekit.NewConfig(node.Light, ip, trustedPeers, bridgeNode.TrustedHash)
	cfg.Core.IP = appNode.IP.To4().String()
	cfg.Core.RPCPort = "26657"
	cfg.Core.GRPCPort = "9090"

	nd, err := nodekit.NewNode(
		ndhome,
		node.Light,
		cfg,
	)
	if err != nil {
		return err
	}

	err = nd.Start(ctx)
	if err != nil {
		return err
	}

	addr, err := nd.StateServ.AccountAddress(ctx)
	if err != nil {
		return err
	}

	_, err = syncclient.PublishAndWait(
		ctx,
		testkit.FundAccountTopic,
		addr.String(),
		testkit.AccountsFundedState,
		runenv.IntParam("validator"),
	)
	if err != nil {
		return err
	}

	eh, err := nd.HeaderServ.GetByHeight(ctx, uint64(runenv.IntParam("block-height")))
	if err != nil {
		return err
	}
	runenv.RecordMessage("Reached Block#%d contains Hash: %s",
		runenv.IntParam("block-height"),
		eh.Commit.BlockID.Hash.String())

	if nd.HeaderServ.IsSyncing() {
		runenv.RecordFailure(fmt.Errorf("full node is still syncing the past"))
	}

	bal, err := nd.StateServ.Balance(ctx)
	if err != nil {
		return err
	}
	if bal.IsZero() {
		return fmt.Errorf("light has no money in the bank")
	}

	runenv.RecordMessage("light -> %d has this %s balance", initCtx.GroupSeq, bal.String())

	nid, _ := hex.DecodeString("52fdfc072182654f")
	data := []byte("163f5f0f9a62037c4d7bbb0407d1e2c64981855ad8681d0d86d1e91e00167939cb6694d2c422acd208a0072939487f")
	for i := 0; i < 10; i++ {
		tx, err := nd.StateServ.SubmitPayForData(ctx, nid, data, 70000)
		if err != nil {
			return err
		}

		runenv.RecordMessage("code reponse is %d", tx.Code)
		runenv.RecordMessage(tx.RawLog)
		if tx.Code != 0 {
			return fmt.Errorf("failed pfd")
		}
	}

	err = nd.Stop(ctx)
	if err != nil {
		return err
	}
	_, err = syncclient.SignalEntry(ctx, testkit.FinishState)
	if err != nil {
		return err
	}

	return err
}