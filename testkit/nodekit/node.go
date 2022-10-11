package nodekit

import (
	"fmt"
	"net"

	"github.com/celestiaorg/celestia-app/app"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/celestiaorg/celestia-node/logs"
	"github.com/celestiaorg/celestia-node/node"
	"github.com/celestiaorg/celestia-node/params"
	logging "github.com/ipfs/go-log/v2"
)

func NewNode(path string, tp node.Type, IP net.IP, trustedHash string, options ...node.Option) (*node.Node, error) {
	// This is necessary to ensure that the account addresses are correctly prefixed
	// as in the celestia application.
	sdkcfg := sdk.GetConfig()
	sdkcfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	sdkcfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	sdkcfg.Seal()
	err := node.Init(path, tp)
	if err != nil {
		return nil, err
	}
	store, err := node.OpenStore(path)
	if err != nil {
		return nil, err
	}

	cfg := node.DefaultConfig(tp)
	cfg.Core.GRPCPort = "9090"
	cfg.P2P.ListenAddresses = []string{fmt.Sprintf("/ip4/%s/tcp/2121", IP)}

	options = append([]node.Option{node.WithConfig(cfg), node.WithNetwork(params.Private), node.WithTrustedHash(trustedHash)}, options...)
	return node.New(tp, store, options...)
}

func SetLoggersLevel(lvl string) error {
	level, err := logging.LevelFromString(lvl)
	if err != nil {
		return err
	}
	logs.SetAllLoggers(level)

	return nil
}
